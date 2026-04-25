#!/bin/bash
# Markpost Development Environment Startup Script
#
# Usage:
#   ./dev.sh              # Start all services
#   ./dev.sh --clean      # Stop and clean up everything
#   ./dev.sh --backend    # Start backend only
#   ./dev.sh --frontend   # Start frontend only

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info() { echo -e "${BLUE}  [info]${NC} $1"; }
success() { echo -e "${GREEN}  [success]${NC} $1"; }
warn() { echo -e "${YELLOW}  [warn]${NC} $1"; }
error() { echo -e "${RED}  [error]${NC} $1"; }

get_worktree_name() {
    local git_dir
    git_dir=$(git rev-parse --git-dir 2>/dev/null) || echo "main"

    if [[ "$git_dir" =~ .*\.git/worktrees/(.+)$ ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        local branch
        branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null) || echo "main"
        echo "$branch"
    fi
}

calculate_port_offset() {
    local name="$1"

    if [[ "$name" == "main" || "$name" == "master" ]]; then
        echo 0
        return
    fi

    local hash
    hash=$(echo -n "$name" | md5sum | cut -c1-6)
    echo $(( 16#$hash % 100 + 1 ))
}

generate_env() {
    info "Generating environment configuration..."

    local worktree_name
    worktree_name=$(get_worktree_name)

    local offset
    offset=$(calculate_port_offset "$worktree_name")
    local sanitized=$(echo "$worktree_name" | tr '.' '-' | tr '[:upper:]' '[:lower:]')
    local project_name="markpost-${sanitized}"

    local backend_port=$((7330 + offset * 10))
    local frontend_port=$((3034 + offset * 10))

    cat > "${SCRIPT_DIR}/.env" << EOF
# Markpost Development Environment
# Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
# Worktree: ${worktree_name}
# Offset: ${offset}

COMPOSE_PROJECT_NAME=${project_name}

MARKPOST_SERVER__PORT=${backend_port}
FRONTEND_PORT=${frontend_port}

MARKPOST_DEBUG=true
MARKPOST_DB__DSN=file:/app/data/markpost.db?_foreign_keys=on&_journal_mode=WAL
MARKPOST_JWT__ACCESS_SIGNING_KEY=dev-access-secret-key-change-in-production
MARKPOST_JWT__REFRESH_SIGNING_KEY=dev-refresh-secret-key-change-in-production
MARKPOST_ADMIN__INITIAL_USERNAME=markpost
MARKPOST_ADMIN__INITIAL_PASSWORD=markpost
EOF

    success "Environment configuration generated"
    info "  Worktree: ${worktree_name}"
    info "  Offset: ${offset}"
    info "  Backend Port: ${backend_port}"
    info "  Frontend Port: ${frontend_port}"
}

generate_frontend_env() {
    info "Generating frontend environment configuration..."

    set -a
    source "${SCRIPT_DIR}/.env"
    set +a

    cat > "${PROJECT_ROOT}/frontend/.env.local" << EOF
# Markpost Frontend Environment
# Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

MARKPOST_SERVER__PORT=${MARKPOST_SERVER__PORT}
EOF

    success "Frontend environment configuration generated"
}

start_docker_services() {
    info "Starting Docker services..."

    cd "${SCRIPT_DIR}"
    docker compose up -d --build

    success "Docker services started"
}

stop_docker_services() {
    info "Stopping Docker services..."

    cd "${SCRIPT_DIR}"
    docker compose down

    success "Docker services stopped"
}

wait_for_backend() {
    set -a
    source "${SCRIPT_DIR}/.env"
    set +a

    info "Waiting for backend to be ready..."
    local max_attempts=60
    local attempt=0

    while [[ $attempt -lt $max_attempts ]]; do
        if curl -s "http://localhost:${MARKPOST_SERVER__PORT}/health" &>/dev/null; then
            success "Backend is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    warn "Backend not ready after $(( max_attempts * 2 )) seconds, but continuing..."
}

start_frontend() {
    info "Starting frontend..."

    set -a
    source "${SCRIPT_DIR}/.env"
    set +a

    cd "${PROJECT_ROOT}/frontend"

    if lsof -Pi :${FRONTEND_PORT} -sTCP:LISTEN -t >/dev/null 2>&1; then
        warn "Port ${FRONTEND_PORT} is already in use"
        return 0
    fi

    if ! command -v pnpm &>/dev/null; then
        error "pnpm is not installed"
        return 1
    fi

    if [[ ! -d "node_modules" ]]; then
        info "Installing dependencies..."
        pnpm install
    fi

    info "Starting Next.js dev server..."
    setsid pnpm dev > "${SCRIPT_DIR}/frontend.log" 2>&1 &
    echo $! > "${SCRIPT_DIR}/frontend.pid"

    local max_attempts=30
    local attempt=0

    while [[ $attempt -lt $max_attempts ]]; do
        if curl -s "http://localhost:${FRONTEND_PORT}" &>/dev/null; then
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
    done

    success "Frontend started"
}

stop_frontend() {
    source "${SCRIPT_DIR}/.env" 2>/dev/null || true
    local port="${FRONTEND_PORT:-3034}"

    if [[ -f "${SCRIPT_DIR}/frontend.pid" ]]; then
        local pid
        pid=$(cat "${SCRIPT_DIR}/frontend.pid")
        if kill -0 "$pid" 2>/dev/null; then
            info "Stopping frontend (pid: $pid)..."
            kill -TERM -- -"$pid" 2>/dev/null || true
            sleep 1
            kill -KILL -- -"$pid" 2>/dev/null || true
        fi
        rm -f "${SCRIPT_DIR}/frontend.pid"
    fi

    local ports=("$port")
    [[ "$port" != "3034" ]] && ports+=(3034)

    for try_port in "${ports[@]}"; do
        if lsof -i :"$try_port" &>/dev/null; then
            info "Stopping frontend on port $try_port..."
            lsof -ti :"$try_port" | xargs kill -9 2>/dev/null || true
        fi
    done

    success "Frontend stopped"
}

display_results() {
    source "${SCRIPT_DIR}/.env"

    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  Markpost Development Environment${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${CYAN}Services:${NC}"
    echo -e "  Frontend:       ${YELLOW}http://localhost:${FRONTEND_PORT}${NC}"
    echo -e "  Backend API:    ${YELLOW}http://localhost:${MARKPOST_SERVER__PORT}/api/v1${NC}"
    echo -e "  Swagger Docs:   ${YELLOW}http://localhost:${MARKPOST_SERVER__PORT}/swagger${NC}"
    echo ""
    echo -e "${CYAN}Admin Account:${NC}"
    echo -e "  Username:       ${YELLOW}markpost${NC} / ${YELLOW}markpost${NC}"
    echo ""
    echo -e "${CYAN}Commands:${NC}"
    echo -e "  View backend logs:  ${YELLOW}docker compose -f deploy/dev/docker-compose.yml logs -f backend${NC}"
    echo -e "  View frontend logs: ${YELLOW}cat deploy/dev/frontend.log${NC}"
    echo -e "  Stop all services:  ${YELLOW}./deploy/dev/dev.sh --clean${NC}"
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
}

cleanup() {
    info "Stopping all services..."

    stop_docker_services
    stop_frontend

    success "All services stopped"
}

main() {
    local mode="all"

    for arg in "$@"; do
        case $arg in
            --clean)
                cleanup
                exit 0
                ;;
            --backend)
                mode="backend"
                ;;
            --frontend)
                mode="frontend"
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --clean      Stop and clean up all services"
                echo "  --backend    Start backend only"
                echo "  --frontend   Start frontend only"
                echo "  --help, -h   Show this help message"
                echo ""
                echo "Without options, starts all services (backend + frontend)."
                exit 0
                ;;
            *)
                error "Unknown argument: $arg"
                echo "Run '$0 --help' for usage information"
                exit 1
                ;;
        esac
    done

    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  Markpost Development Environment${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""

    generate_env

    if [[ "$mode" == "backend" || "$mode" == "all" ]]; then
        generate_frontend_env
        start_docker_services
        wait_for_backend
    fi

    if [[ "$mode" == "frontend" || "$mode" == "all" ]]; then
        if [[ "$mode" == "frontend" ]]; then
            generate_frontend_env
        fi
        start_frontend
    fi

    display_results
}

main "$@"
