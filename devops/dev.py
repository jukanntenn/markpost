#!/usr/bin/env python3
"""Markpost development environment manager.

All services (backend, frontend, postgres) run in Docker Compose.

Usage:
    python devops/dev.py start    # Start all services (default)
    python devops/dev.py stop     # Stop all services
    python devops/dev.py logs [svc]  # Tail logs (svc: backend|frontend|postgres|'')

Color output is off by default (AI-agent friendly). Pass --color for
human-readable ANSI-colored status lines; NO_COLOR forces it off regardless.
"""

import argparse
import logging
import os
import subprocess
import sys
import time
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent
COMPOSE_FILE = SCRIPT_DIR / "docker-compose.yml"
LOGS_HOST_DIR = SCRIPT_DIR / "data" / "logs"

BACKEND_PORT = 7330
FRONTEND_PORT = 3034

# ANSI escape codes; only emitted when color is enabled (see color_enabled).
_GREEN = "\033[32m"
_YELLOW = "\033[33m"
_RED = "\033[31m"
_RESET = "\033[0m"

logger = logging.getLogger("dev")
color_enabled = False


def setup_logging():
    handler_out = logging.StreamHandler(sys.stdout)
    handler_out.setLevel(logging.INFO)
    handler_out.addFilter(lambda record: record.levelno <= logging.INFO)

    handler_err = logging.StreamHandler(sys.stderr)
    handler_err.setLevel(logging.WARNING)

    logging.basicConfig(
        level=logging.INFO,
        handlers=[handler_out, handler_err],
        format="%(message)s",
    )


def paint(level, fmt, *args):
    """Log at level, prefixing with a colored tag when color is enabled."""
    if not color_enabled:
        logger.log(level, fmt, *args)
        return
    tag_color = {
        logging.INFO: _GREEN,
        logging.WARNING: _YELLOW,
        logging.ERROR: _RED,
    }.get(level, _GREEN)
    tag = {
        logging.INFO: "ok",
        logging.WARNING: "warn",
        logging.ERROR: "fail",
    }.get(level, "ok")
    msg = fmt % args if args else fmt
    logger.log(level, "%s[%s]%s %s", tag_color, tag, _RESET, msg)


def parse_args():
    parser = argparse.ArgumentParser(
        description="Markpost development environment manager"
    )
    parser.add_argument(
        "--color",
        action="store_true",
        help="Enable ANSI color in status output (off by default for AI agents).",
    )
    sub = parser.add_subparsers(dest="command")
    sub.add_parser("start", help="Start all services (default)")
    sub.add_parser("stop", help="Stop all services")
    logs = sub.add_parser("logs", help="Tail service logs")
    logs.add_argument(
        "service", nargs="?", default="", help="backend|frontend|postgres (empty=all)"
    )
    args = parser.parse_args()
    if not args.command:
        args.command = "start"
    return args


def run(name, *args, **kwargs):
    defaults = {"stdout": subprocess.PIPE, "stderr": subprocess.PIPE, "text": True}
    defaults.update(kwargs)
    return subprocess.run([name, *args], **defaults)


def run_check(name, *args, **kwargs):
    result = run(name, *args, **kwargs)
    if result.returncode != 0:
        paint(logging.ERROR, "%s %s (exit %d)", name, " ".join(args), result.returncode)
        if result.stdout:
            sys.stderr.write(result.stdout)
        if result.stderr:
            sys.stderr.write(result.stderr)
        sys.exit(result.returncode)
    return result


def compose(*args):
    return run_check("docker", "compose", "-f", str(COMPOSE_FILE), *args)


def dump_logs(service):
    """Print a service's container logs to stderr to aid diagnosis."""
    paint(logging.ERROR, "dumping %s logs:", service)
    subprocess.run(
        ["docker", "compose", "-f", str(COMPOSE_FILE), "logs", "--tail=80", service],
        stdout=sys.stderr,
        stderr=sys.stderr,
    )


def wait_for(url, name, timeout=120):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            result = run("curl", "-sf", url)
            if result.returncode == 0:
                paint(logging.INFO, "%s is ready", name)
                return True
        except Exception:
            pass
        time.sleep(2)
    paint(logging.ERROR, "%s not ready after %ds", name, timeout)
    return False


def start():
    paint(logging.INFO, "Starting all services (backend + frontend + postgres)...")

    # Pre-create the host log dir so the bind mount target is owned by the
    # invoking user, not root (docker creates a root-owned dir otherwise, and
    # the non-root backend container could fail to write observability JSONL).
    LOGS_HOST_DIR.mkdir(parents=True, exist_ok=True)

    compose("up", "-d", "--build")

    backend_ok = wait_for(f"http://localhost:{BACKEND_PORT}/api/v1/health", "Backend")
    if not backend_ok:
        dump_logs("backend")
        paint(logging.ERROR, "Backend failed to start; see logs above. Aborting.")
        try:
            compose("down")
        except SystemExit:
            pass
        sys.exit(1)

    frontend_ok = wait_for(f"http://localhost:{FRONTEND_PORT}", "Frontend", timeout=60)
    if not frontend_ok:
        dump_logs("frontend")
        paint(logging.ERROR, "Frontend failed to start; see logs above. Aborting.")
        try:
            compose("down")
        except SystemExit:
            pass
        sys.exit(1)

    paint(logging.INFO, "Services:")
    paint(logging.INFO, "  Frontend:     http://localhost:%d", FRONTEND_PORT)
    paint(logging.INFO, "  Backend API:  http://localhost:%d/api/v1", BACKEND_PORT)
    paint(logging.INFO, "  Swagger Docs: http://localhost:%d/swagger", BACKEND_PORT)
    paint(logging.INFO, "  Logs:         %s  (host mount)", LOGS_HOST_DIR)
    paint(logging.INFO, "Admin: username=markpost password=markpost")
    paint(logging.INFO, "Stop: python devops/dev.py stop")


def stop():
    paint(logging.INFO, "Stopping all services...")
    compose("down")
    paint(logging.INFO, "Stopped.")


def logs(service):
    args = ["logs", "-f"]
    if service:
        args.append(service)
    subprocess.run(["docker", "compose", "-f", str(COMPOSE_FILE), *args])


def main():
    global color_enabled
    setup_logging()
    args = parse_args()
    # Color is opt-in (--color) and suppressed entirely when NO_COLOR is set
    # (https://no-color.org/), so AI agents get plain text by default.
    color_enabled = args.color and "NO_COLOR" not in os.environ
    if args.command == "start":
        start()
    elif args.command == "stop":
        stop()
    elif args.command == "logs":
        logs(args.service)


if __name__ == "__main__":
    main()
