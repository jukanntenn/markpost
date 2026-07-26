#!/usr/bin/env python3
"""Markpost development environment manager.

All services (backend, frontend, postgres) run in Docker Compose.
Usage:
    python devops/dev.py start    # Start all services (default)
    python devops/dev.py stop     # Stop all services
    python devops/dev.py logs [svc]  # Tail logs (svc: backend|frontend|postgres|'')
"""

import argparse
import logging
import subprocess
import sys
import time
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent
COMPOSE_FILE = SCRIPT_DIR / "docker-compose.yml"

BACKEND_PORT = 7330
FRONTEND_PORT = 3034

logger = logging.getLogger("dev")


def setup_logging():
    handler_out = logging.StreamHandler(sys.stdout)
    handler_out.setLevel(logging.INFO)
    handler_out.add_filter(lambda record: record.levelno <= logging.INFO)
    handler_err = logging.StreamHandler(sys.stderr)
    handler_err.setLevel(logging.WARNING)
    logging.basicConfig(level=logging.INFO, handlers=[handler_out, handler_err])


def parse_args():
    parser = argparse.ArgumentParser(
        description="Markpost development environment manager"
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
        logger.error("[fail] %s %s (exit %d)", name, " ".join(args), result.returncode)
        if result.stdout:
            logger.error("%s", result.stdout)
        if result.stderr:
            logger.error("%s", result.stderr)
        sys.exit(result.returncode)
    return result


def compose(*args):
    return run_check("docker", "compose", "-f", str(COMPOSE_FILE), *args)


def wait_for(url, name, timeout=120):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            result = run("curl", "-sf", url)
            if result.returncode == 0:
                logger.info("[ok] %s is ready", name)
                return True
        except Exception:
            pass
        time.sleep(2)
    logger.warning("[warn] %s not ready after %ds, continuing...", name, timeout)
    return False


def start():
    logger.info("Starting all services (backend + frontend + postgres)...")
    compose("up", "-d", "--build")
    wait_for(f"http://localhost:{BACKEND_PORT}/api/v1/health", "Backend")
    wait_for(f"http://localhost:{FRONTEND_PORT}", "Frontend", timeout=60)
    logger.info("Services:")
    logger.info("  Frontend:     http://localhost:%d", FRONTEND_PORT)
    logger.info("  Backend API:  http://localhost:%d/api/v1", BACKEND_PORT)
    logger.info("  Swagger Docs: http://localhost:%d/swagger", BACKEND_PORT)
    logger.info("  Logs:         devops/data/logs/  (host mount)")
    logger.info("Admin: username=markpost password=markpost")
    logger.info("Stop: python devops/dev.py stop")


def stop():
    logger.info("Stopping all services...")
    compose("down")
    logger.info("Stopped.")


def logs(service):
    args = ["logs", "-f"]
    if service:
        args.append(service)
    subprocess.run(["docker", "compose", "-f", str(COMPOSE_FILE), *args])


def main():
    setup_logging()
    args = parse_args()
    if args.command == "start":
        start()
    elif args.command == "stop":
        stop()
    elif args.command == "logs":
        logs(args.service)


if __name__ == "__main__":
    main()
