#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "$SCRIPT_DIR"

if [ ! -f .env ]; then
    cat > .env <<EOF
MARKPOST_SERVER__PORT=7330
POSTGRES_PORT=5432
EOF
    echo "Created .env file"
fi

echo "Starting development environment..."

docker compose up -d postgres

echo "Waiting for PostgreSQL to be ready..."
sleep 5

until docker compose exec -T postgres pg_isready -U markpost -d markpost; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done

echo "PostgreSQL is ready!"

docker compose up -d backend

echo ""
echo "✅ Development environment started!"
echo "🌐 API: http://localhost:${MARKPOST_SERVER__PORT:-7330}"
echo "📊 PostgreSQL: localhost:${POSTGRES_PORT:-5432}"
echo ""
echo "To view logs: docker compose logs -f backend"
echo "To stop: docker compose down"
