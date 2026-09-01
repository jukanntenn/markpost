#!/usr/bin/env python3
"""Render devops/ansible/templates/*.j2 and verify the output still parses.

prek pre-commit hook (ansible-templates).

Why this exists: a formatter (djlint) used to rewrite these templates and
collapsed line-leading `{% if %}` / `{% endif %}` into the neighbouring line.
Ansible's template module renders with trim_blocks=True, which only strips the
newline after a block tag when that tag ends the line — so a collapsed tag
renders as `{% if x %}key = "v"` on one line and silently produces invalid
TOML/YAML. Nothing caught it until a deploy failed.

Every conditional branch is rendered, because the damage only shows up on one
side of an `{% if %}`.
"""

from __future__ import annotations

from pathlib import Path
import subprocess
import sys

try:
    import jinja2
    import yaml
except ImportError as exc:  # pragma: no cover - environment-dependent
    print(f"[ansible-templates] {exc.name} not installed, skipping")
    sys.exit(0)

import tomllib

# Mirrors ansible.builtin.template defaults.
JINJA_KWARGS = {
    "trim_blocks": True,
    "lstrip_blocks": False,
    "keep_trailing_newline": True,
    "undefined": jinja2.StrictUndefined,
}

BASE_VARS = {
    "ansible_managed": "Ansible managed",
    "app_name": "markpost",
    "app_path": "/home/deploy/docker/markpost",
    "caddy_port": 2053,
    "caddyfile": "/home/deploy/docker/markpost/Caddyfile",
    "cloudflare_cidrs": "173.245.48.0/20 2400:cb00::/32",
    "config_file": "/home/deploy/docker/markpost/config.toml",
    "db_password": "pw",
    "db_timezone": "UTC",
    "debug": False,
    "go_port": 7330,
    "host_port": 8089,
    "tls_profile": "http",
    "gateway_host": "markpost.example.com",
    "gateway_certs_dir": "/etc/caddy/certs/markpost",
    "image": "jukanntenn/markpost:v0.0.0",
    "jwt_access_signing_key": "a",
    "jwt_refresh_signing_key": "r",
    "admin_password": "p",
    "postgres_db": "markpost",
    "postgres_user": "markpost",
    # host_vars fact every inventory host defines (app_path above already
    # assumes it); the heartbeat conf interpolates the vaulted push URL into
    # its environment= line — the deploy tasks guard on the variable, the
    # conf template assumes it exists.
    "user": "deploy",
    "kuma_heartbeat_url": "https://kuma.example.com/api/push/token",
    # Same contract for the Beszel agent compose: the deploy tasks guard on
    # beszel_hub_url (set in group_vars/production once the ops hub exists),
    # the template assumes all three exist.
    "beszel_agent_image": "henrygd/beszel-agent:0.0.0",
    "beszel_agent_key": "ssh-ed25519-AAAAC3Nzatest-render-key",
    "beszel_hub_url": "https://beszel.example.com/beszel/agent",
}

SCENARIOS = {
    "dev (no public_url, no oauth, no cloudflare)": {
        "env": "dev",
    },
    "staging (public_url, no oauth)": {
        "env": "staging",
        "public_url": "https://markpost.example.com",
        "trusted_proxies": '["127.0.0.1", "::1"]',
    },
    "staging (public_url + oauth)": {
        "env": "staging",
        "public_url": "https://markpost.example.com",
        "github_client_id": "cid",
        "github_client_secret": "secret",
    },
    "production (oauth + cloudflare + certs)": {
        "env": "production",
        "public_url": "https://markpost.example.com",
        "github_client_id": "cid",
        "github_client_secret": "secret",
        "cloudflare_api_token": "token",
        "cloudflare_zone_id": "zone",
        "retention_days": 30,
        "delivery_history_retention": "720h",
    },
    # An oauth id without a public_url must not emit a half-configured
    # redirect_url — the guard is `and public_url is defined`.
    "oauth id but no public_url": {
        "env": "staging",
        "github_client_id": "cid",
        "github_client_secret": "secret",
    },
}


def check_config(doc: dict, scenario: dict, fail) -> None:
    want_public_url = "public_url" in scenario
    has_public_url = "public_url" in doc["server"]
    if has_public_url is not want_public_url:
        fail(f"[server].public_url present={has_public_url}, want={want_public_url}")

    want_oauth = want_public_url and bool(scenario.get("github_client_id"))
    has_oauth = bool(doc["oauth"]["github"]["client_id"])
    if has_oauth is not want_oauth:
        fail(f"[oauth.github].client_id set={has_oauth}, want={want_oauth}")
    if want_oauth and not doc["oauth"]["github"].get("redirect_url"):
        fail("[oauth.github].redirect_url missing while oauth is configured")

    want_cf = bool(scenario.get("cloudflare_api_token"))
    if ("cloudflare" in doc) is not want_cf:
        fail(f"[cloudflare] present={'cloudflare' in doc}, want={want_cf}")

    if doc["db"]["driver"] != "postgresql":
        fail("[db].driver must be postgresql")
    # Both DSN forms work at runtime (MigrateUp opens the pool with lib/pq, which
    # takes either), so this guards escaping rather than parseability: the
    # password comes from the vault and is unconstrained, and the key-value form
    # would need it single-quoted and backslash-escaped, whereas the URL form
    # only needs the percent-encoding `| urlencode` already applies.
    dsn = doc["db"]["dsn"]
    if not dsn.startswith(("postgres://", "postgresql://")):
        fail(
            f"[db].dsn must be URL form so the password is percent-encoded, got {dsn[:24]!r}..."
        )
    if "password=" in dsn and " " in dsn:
        fail("[db].dsn looks like the libpq key-value form")
    if not doc["observability"]["log_dir"].startswith("/app/data"):
        fail("[observability].log_dir must live under the /app/data bind-mount")


def check_compose(doc: dict, scenario: dict, fail) -> None:
    mounts = doc["services"]["markpost"]["volumes"]
    # No environment mounts certs into the container any more — TLS moved to
    # the host Caddy gateway (the playbook copies the cert to /etc/caddy).
    if any("/app/certs" in m for m in mounts):
        fail("certs mount present; TLS terminates on the host gateway now")
    if not any(m.endswith("/app/data") for m in mounts):
        fail("/app/data bind-mount missing")
    for name in ("markpost", "postgres"):
        if "logging" not in doc["services"][name]:
            fail(f"service {name} has no logging cap")


def parse_ini(text: str) -> dict:
    import configparser
    import io

    parser = configparser.ConfigParser()
    parser.read_string(text)
    return {s: dict(parser[s]) for s in parser.sections()}


def check_heartbeat_conf(doc: dict, scenario: dict, fail) -> None:
    program = doc.get("program:markpost-heartbeat")
    if program is None:
        fail("supervisor section [program:markpost-heartbeat] missing")
        return
    if program.get("autorestart") != "true":
        fail("heartbeat program must autorestart")
    env = program.get("environment", "")
    if "KUMA_HEARTBEAT_URL=" not in env:
        # The script refuses to start without it; a conf that lost the
        # environment line would look fine until the first deploy.
        fail("environment= must carry KUMA_HEARTBEAT_URL for the heartbeat")


CHECKS = {
    "config.toml.j2": (tomllib.loads, check_config),
    "docker-compose.yml.j2": (yaml.safe_load, check_compose),
    "markpost-heartbeat.conf.j2": (parse_ini, check_heartbeat_conf),
}


def main() -> int:
    root = Path(
        subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True,
            text=True,
            check=True,
        ).stdout.strip()
    )
    templates = root / "devops" / "ansible" / "templates"

    env = jinja2.Environment(loader=jinja2.FileSystemLoader(templates), **JINJA_KWARGS)

    failures: list[str] = []
    for path in sorted(templates.glob("*.j2")):
        parse, check = CHECKS.get(path.name, (None, None))
        for label, overrides in SCENARIOS.items():

            def fail(msg: str, *, _p=path.name, _l=label) -> None:
                failures.append(f"{_p} [{_l}]: {msg}")

            try:
                rendered = env.get_template(path.name).render(BASE_VARS | overrides)
            except jinja2.UndefinedError as exc:
                fail(f"undefined variable: {exc.message}")
                continue

            if parse is None:
                continue
            try:
                doc = parse(rendered)
            except Exception as exc:
                fail(f"rendered output does not parse: {exc}")
                continue
            check(doc, BASE_VARS | overrides, fail)

    if failures:
        print("ERROR: ansible template render check failed", file=sys.stderr)
        for f in failures:
            print(f"  {f}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
