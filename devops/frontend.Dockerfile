FROM node:24-alpine

RUN apk add --no-cache git && corepack enable

WORKDIR /app

ENV NEXT_TELEMETRY_DISABLED=1

# Install deps with the lockfile first for better layer caching. The tree is
# chowned to uid 1000 because the container runs as that user (compose user:
# "${UID:-1000}"): pnpm may need to recreate node_modules at runtime, and a
# root-owned tree makes that purge fail with EACCES.
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile \
    && mkdir -p /app/.next \
    && chown -R 1000:1000 /app/node_modules /app/.next

EXPOSE 3034

CMD ["pnpm", "dev"]
