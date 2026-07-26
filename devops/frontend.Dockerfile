FROM node:24-alpine

RUN apk add --no-cache git && corepack enable

WORKDIR /app

ENV NEXT_TELEMETRY_DISABLED=1

# Install deps with the lockfile first for better layer caching.
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

EXPOSE 3034

CMD ["pnpm", "dev"]
