#!/usr/bin/env python3
"""Build multi-architecture Docker images for markpost using Docker buildx.

Builds two images: markpost (backend) and markpost-web (frontend).
Supports load (local, single platform) and push (multi-platform to registry) modes.
Always applies a 'dev' tag; additional tags can be specified via --tags.
Automatically registers QEMU binfmt for cross-platform builds when needed.
"""

import argparse
import logging
import os
import platform
import subprocess
import sys
import time

IMAGE_NAME = "markpost"
IMAGE_NAME_WEB = "markpost-web"
DEFAULT_REGISTRY = "192.168.5.50:5000"
PLATFORMS = ("linux/amd64", "linux/arm64")

BACKEND_DOCKERFILE = "docker/backend.Dockerfile"
FRONTEND_DOCKERFILE = "docker/frontend.Dockerfile"

QEMU_ARCH_MAP = {
    "linux/arm64": "arm64",
}

logger = logging.getLogger("build")


def setup_logging():
    handler_out = logging.StreamHandler(sys.stdout)
    handler_out.setLevel(logging.INFO)
    handler_out.addFilter(lambda record: record.levelno <= logging.INFO)

    handler_err = logging.StreamHandler(sys.stderr)
    handler_err.setLevel(logging.WARNING)

    logging.basicConfig(level=logging.INFO, handlers=[handler_out, handler_err])


def parse_args():
    parser = argparse.ArgumentParser(
        description="Build multi-architecture Docker images"
    )
    parser.add_argument(
        "--push",
        action="store_true",
        help="Push images to registry (default: load locally)",
    )
    parser.add_argument(
        "--registry",
        default=DEFAULT_REGISTRY,
        help=f"Container registry (default: {DEFAULT_REGISTRY})",
    )
    parser.add_argument(
        "--tags",
        nargs="+",
        action="extend",
        default=[],
        help="Additional image tags (dev tag is always applied)",
    )
    parser.add_argument(
        "--backend-only", action="store_true", help="Build only the backend image"
    )
    parser.add_argument(
        "--frontend-only", action="store_true", help="Build only the frontend image"
    )
    return parser.parse_args()


def detect_host_platform():
    machine = platform.machine().lower()
    if machine in ("x86_64", "amd64"):
        return "linux/amd64"
    if machine in ("aarch64", "arm64"):
        return "linux/arm64"
    return "linux/amd64"


def is_qemu_registered(arch):
    return os.path.exists(f"/proc/sys/fs/binfmt_misc/qemu-{arch}")


def ensure_qemu(foreign_platforms):
    missing_archs = []
    for p in foreign_platforms:
        arch = QEMU_ARCH_MAP.get(p)
        if arch and not is_qemu_registered(arch):
            missing_archs.append(arch)

    if not missing_archs:
        return

    arch_list = ", ".join(missing_archs)
    logger.info(
        "QEMU binfmt not registered for: %s. Registering via tonistiigi/binfmt...",
        arch_list,
    )

    try:
        subprocess.run(
            [
                "docker",
                "run",
                "--rm",
                "--privileged",
                "tonistiigi/binfmt",
                "--install",
                arch_list,
            ],
            check=True,
        )
    except subprocess.CalledProcessError as e:
        logger.error("Failed to register QEMU binfmt (exit code %d).", e.returncode)
        logger.error(
            "Try running manually: docker run --rm --privileged tonistiigi/binfmt --install %s",
            arch_list,
        )
        sys.exit(1)

    for arch in missing_archs:
        registered = False
        for attempt in range(5):
            if is_qemu_registered(arch):
                registered = True
                break
            time.sleep(1)
        if not registered:
            logger.error(
                "QEMU registration for %s did not take effect after retries.",
                arch,
            )
            sys.exit(1)

    logger.info("QEMU binfmt registered successfully for: %s", arch_list)


def parse_builder_info(inspect_output):
    driver = "unknown"
    builder_platforms = []
    builder_name = "unknown"

    for line in inspect_output.splitlines():
        stripped = line.strip()
        if stripped.startswith("Name:") and builder_name == "unknown":
            builder_name = stripped.split(":", 1)[1].strip()
        elif stripped.startswith("Driver:"):
            driver = stripped.split(":", 1)[1].strip()
        elif stripped.startswith("Platforms:"):
            builder_platforms = [
                p.strip() for p in stripped.split(":", 1)[1].split(",")
            ]

    return builder_name, driver, builder_platforms


def check_buildx(target_platforms):
    try:
        subprocess.run(
            ["docker", "buildx", "version"],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        logger.error(
            "Docker buildx is not available. Ensure Docker is installed and buildx plugin is enabled."
        )
        sys.exit(1)

    result = subprocess.run(
        ["docker", "buildx", "inspect"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        logger.error(
            "Failed to inspect active buildx builder: %s", result.stderr.strip()
        )
        sys.exit(1)

    builder_name, driver, builder_platforms = parse_builder_info(result.stdout)
    missing = [p for p in target_platforms if p not in builder_platforms]

    if missing:
        host_platform = detect_host_platform()
        foreign_platforms = [p for p in missing if p != host_platform]

        if foreign_platforms:
            ensure_qemu(foreign_platforms)

            logger.info("Bootstrapping builder to detect new platforms...")
            bootstrap = subprocess.run(
                ["docker", "buildx", "inspect", "--bootstrap"],
                capture_output=True,
                text=True,
            )
            if bootstrap.returncode != 0:
                logger.error(
                    "Failed to bootstrap builder: %s", bootstrap.stderr.strip()
                )
                sys.exit(1)

            result = subprocess.run(
                ["docker", "buildx", "inspect"],
                capture_output=True,
                text=True,
            )
            builder_name, driver, builder_platforms = parse_builder_info(result.stdout)
            missing = [p for p in target_platforms if p not in builder_platforms]

    if missing:
        logger.error(
            "Active builder does not support required platform(s): %s\n"
            "  Builder:           %s\n"
            "  Builder driver:    %s\n"
            "  Builder platforms: %s\n"
            "  Required:          %s",
            ", ".join(missing),
            builder_name,
            driver,
            ", ".join(builder_platforms) or "(none detected)",
            ", ".join(target_platforms),
        )
        sys.exit(1)

    return builder_name, driver


def build_single_image(args, image_name, dockerfile, context_subdir):
    all_tags = ["dev"] + args.tags

    if args.push:
        target_platforms = list(PLATFORMS)
        mode = "push"
    else:
        target_platforms = [detect_host_platform()]
        mode = "load"

    builder_name, driver = check_buildx(target_platforms)

    project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    dockerfile_path = os.path.join(project_root, dockerfile)
    context_path = os.path.join(project_root, context_subdir)

    full_image_names = []
    cmd = ["docker", "buildx", "build"]
    for tag in all_tags:
        if args.push:
            full_tag = f"{args.registry}/{image_name}:{tag}"
        else:
            full_tag = f"{image_name}:{tag}"
        full_image_names.append(full_tag)
        cmd.extend(["--tag", full_tag])

    cmd.extend(["--platform", ",".join(target_platforms)])

    if args.push:
        cmd.append("--push")
    else:
        cmd.append("--load")

    cmd.extend(["-f", dockerfile_path, context_path])

    logger.info("Build configuration (%s):", image_name)
    logger.info("  Mode:       %s", mode)
    logger.info("  Platforms:  %s", ", ".join(target_platforms))
    logger.info("  Builder:    %s (driver: %s)", builder_name, driver)
    logger.info("  Images:     %s", ", ".join(full_image_names))
    logger.info("  Context:    %s", context_path)
    logger.info("  Dockerfile: %s", dockerfile_path)

    try:
        subprocess.run(cmd, check=True)
    except subprocess.CalledProcessError as e:
        logger.error("Build failed for %s (exit code %d)!", image_name, e.returncode)
        sys.exit(1)

    logger.info("Build completed for %s: %s", image_name, ", ".join(full_image_names))


def main():
    setup_logging()
    args = parse_args()

    if args.backend_only and args.frontend_only:
        logger.error("Cannot specify both --backend-only and --frontend-only")
        sys.exit(1)

    if not args.frontend_only:
        build_single_image(args, IMAGE_NAME, BACKEND_DOCKERFILE, "backend")

    if not args.backend_only:
        build_single_image(args, IMAGE_NAME_WEB, FRONTEND_DOCKERFILE, "frontend")


if __name__ == "__main__":
    main()
