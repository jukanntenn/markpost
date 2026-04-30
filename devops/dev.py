#!/usr/bin/env python3
"""Markpost development environment manager.

Manages backend (Docker Compose) and frontend (local pnpm) for local development.

Usage:
    python devops/dev.py start    # Start all services (default)
    python devops/dev.py stop     # Stop all services
"""

import argparse
import logging
import os
import signal
import subprocess
import sys
import time
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent
ENV_FILE = SCRIPT_DIR / ".env"
PID_FILE = SCRIPT_DIR / "frontend.pid"
LOG_FILE = SCRIPT_DIR / "frontend.log"
COMPOSE_FILE = SCRIPT_DIR / "docker-compose.yml"

BACKEND_PORT = 7330
FRONTEND_PORT = 3034

logger = logging.getLogger("dev")


def setup_logging():
    handler_out = logging.StreamHandler(sys.stdout)
    handler_out.setLevel(logging.INFO)
    handler_out.addFilter(lambda record: record.levelno <= logging.INFO)

    handler_err = logging.StreamHandler(sys.stderr)
    handler_err.setLevel(logging.WARNING)

    logging.basicConfig(level=logging.INFO, handlers=[handler_out, handler_err])


def parse_args():
    parser = argparse.ArgumentParser(description="Markpost development environment manager")
    sub = parser.add_subparsers(dest="command")
    sub.add_parser("start", help="Start all services (default)")
    sub.add_parser("stop", help="Stop all services")
    args = parser.parse_args()
    if not args.command:
        args.command = "start"
    return args


def load_env():
    global BACKEND_PORT, FRONTEND_PORT
    if not ENV_FILE.exists():
        return
    for line in ENV_FILE.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        if key == "MARKPOST_SERVER__PORT":
            BACKEND_PORT = int(value)
        elif key == "FRONTEND_PORT":
            FRONTEND_PORT = int(value)


def run(name, *args, **kwargs):
    defaults = {"stdout": subprocess.PIPE, "stderr": subprocess.PIPE, "text": True}
    defaults.update(kwargs)
    return subprocess.run([name, *args], **defaults)


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
    load_env()
    logger.info("Starting backend (Docker Compose)...")
    run("docker", "compose", "-f", str(COMPOSE_FILE), "up", "-d", "--build")
    logger.info("Docker services started")

    wait_for(f"http://localhost:{BACKEND_PORT}/api/v1/health", "Backend")

    frontend_dir = PROJECT_ROOT / "frontend"
    if not (frontend_dir / "node_modules").exists():
        logger.info("Installing frontend dependencies...")
        run("pnpm", "install", cwd=str(frontend_dir))

    logger.info("Starting frontend (pnpm dev)...")
    with open(LOG_FILE, "w") as log:
        proc = subprocess.Popen(
            ["pnpm", "dev"],
            cwd=str(frontend_dir),
            stdout=log,
            stderr=log,
            start_new_session=True,
        )
    PID_FILE.write_text(str(proc.pid))

    wait_for(f"http://localhost:{FRONTEND_PORT}", "Frontend", timeout=60)

    logger.info("Services:")
    logger.info("  Frontend:     http://localhost:%d", FRONTEND_PORT)
    logger.info("  Backend API:  http://localhost:%d/api/v1", BACKEND_PORT)
    logger.info("  Swagger Docs: http://localhost:%d/swagger", BACKEND_PORT)
    logger.info("  PostgreSQL:   localhost:%s", os.environ.get("POSTGRES_PORT", "5432"))
    logger.info("Admin:")
    logger.info("  Username:     markpost")
    logger.info("  Password:     markpost")
    logger.info("Stop: python devops/dev.py stop")


def stop():
    load_env()
    logger.info("Stopping backend...")
    run("docker", "compose", "-f", str(COMPOSE_FILE), "down")
    logger.info("Docker services stopped")

    if PID_FILE.exists():
        pid = int(PID_FILE.read_text().strip())
        try:
            os.killpg(pid, signal.SIGTERM)
            time.sleep(1)
            os.killpg(pid, signal.SIGKILL)
        except (ProcessLookupError, PermissionError):
            pass
        PID_FILE.unlink()

    logger.info("Stopping frontend...")
    try:
        result = run("lsof", "-ti", f":{FRONTEND_PORT}")
        if result.stdout.strip():
            for pid_str in result.stdout.strip().splitlines():
                subprocess.run(["kill", "-9", pid_str], capture_output=True)
    except Exception:
        pass
    logger.info("Frontend stopped")


def main():
    setup_logging()
    args = parse_args()
    if args.command == "start":
        start()
    elif args.command == "stop":
        stop()


if __name__ == "__main__":
    main()
