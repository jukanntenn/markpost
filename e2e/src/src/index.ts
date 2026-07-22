import { dag, Container, Directory, Service, object, func, ReturnType } from "@dagger.io/dagger"

const DB_USER = "markpost"
const DB_PASSWORD = "markpost"
const ADMIN_USERNAME = "markpost"
const ADMIN_PASSWORD = "markpost"
const ACCESS_SIGNING_KEY = "e2e-access-secret-key-min-32-characters!!"
const REFRESH_SIGNING_KEY = "e2e-refresh-secret-key-min-32-characters!!"

function buildConfigToml(dbHost: string, dbName: string, oauthMockHost: string): string {
  return `
[server]
host = "0.0.0.0"
port = 7330
public_url = "https://app:2053"

[db]
driver = "postgresql"
dsn = "postgres://${DB_USER}:${DB_PASSWORD}@${dbHost}:5432/${dbName}?sslmode=disable"

[admin]
initial_username = "${ADMIN_USERNAME}"
initial_password = "${ADMIN_PASSWORD}"

[jwt]
access_signing_key = "${ACCESS_SIGNING_KEY}"
refresh_signing_key = "${REFRESH_SIGNING_KEY}"

[oauth.github]
client_id = "mock-client-id"
client_secret = "mock-client-secret"
redirect_url = "https://app:2053/auth/callback"
auth_url = "http://${oauthMockHost}:3001/login/oauth/authorize"
token_url = "http://${oauthMockHost}:3001/login/oauth/access_token"
user_url = "http://${oauthMockHost}:3001"

[cors]
allow_origins = ["https://localhost:2053", "https://app:2053"]
allow_headers = ["Origin", "Content-Type", "Authorization"]
expose_headers = ["Content-Length"]

[ratelimit.read]
per_second = 100
burst = 200

[ratelimit.public_write]
per_second = 100
burst = 100
daily_per_second = 100
daily_burst = 1000

[ratelimit.authed_write]
per_second = 100
burst = 100
`.trim()
}

@object()
export class MarkpostE2E {
  @func()
  async health(source: Directory): Promise<string> {
    const checks: string[] = []

    const pg = this.postgresService("markpost_health")
    const pgCheck = dag.container()
      .from("alpine:3.20")
      .withServiceBinding("postgres", pg)
      .withExec(["sh", "-c", "apk add --no-cache postgresql-client >/dev/null 2>&1 && pg_isready -h postgres -U markpost"])
    const pgExit = await pgCheck.exitCode()
    checks.push(`postgres service binding: ${pgExit === 0 ? "ok" : "FAIL"}`)

    const appImg = this.appContainer(source)
    const appImgDigest = await appImg.exitCode()
    checks.push(`app dockerBuild: ${appImgDigest === 0 ? "ok" : "FAIL"}`)

    const runner = this.testRunner(source)
    const runnerExit = await runner.exitCode()
    checks.push(`test runner (npm install): ${runnerExit === 0 ? "ok" : "FAIL"}`)

    const allOk = checks.every((c) => c.endsWith("ok"))
    checks.push(`overall: ${allOk ? "healthy" : "unhealthy"}`)
    if (!allOk) {
      throw new Error(`Dagger e2e health check failed:\n${checks.join("\n")}`)
    }
    return checks.join("\n")
  }

  @func()
  async test(testFile: string, source: Directory): Promise<string> {
    const runId = testFile.replace(/[^a-zA-Z0-9]/g, "_") + "-" + Date.now()
    const dbName = `markpost_${runId}`

    const postgres = this.postgresService(dbName)
    const webhookMock = this.webhookMockService(source, runId)
    const oauthMock = this.oauthMockService(source, runId)
    const configToml = buildConfigToml("postgres", dbName, "oauth-mock")
    const appSvc = this.appService(source, postgres, webhookMock, oauthMock, configToml, runId)

    const runner = this.testRunner(source)
      .withServiceBinding("app", appSvc)
      .withServiceBinding("webhook-mock", webhookMock)
      .withServiceBinding("oauth-mock", oauthMock)
      .withEnvVariable("BASE_URL", "https://app:2053")
      .withEnvVariable("BACKEND_URL", "https://app:2053")
      .withEnvVariable("ADMIN_USERNAME", ADMIN_USERNAME)
      .withEnvVariable("ADMIN_PASSWORD", ADMIN_PASSWORD)
      .withEnvVariable("WEBHOOK_MOCK_URL", "http://webhook-mock:3002")
      .withEnvVariable("OAUTH_MOCK_URL", "http://oauth-mock:3001")
      .withEnvVariable("RUN_ID", runId)
      .withEnvVariable("NODE_TLS_REJECT_UNAUTHORIZED", "0")
      .withExec(["npx", "playwright", "test", "--config=playwright.config.ts", `tests/${testFile}`], { expect: ReturnType.Any })

    const exitCode = await runner.exitCode()
    const stdout = await runner.stdout()
    if (exitCode !== 0) {
      throw new Error(`Test failed: ${testFile}\n${stdout}`)
    }
    return stdout
  }

  @func()
  async all(source: Directory, concurrency: number = 3): Promise<string> {
    const entries = await source.directory("e2e/tests").entries()
    const specs = entries.filter((f: string) => f.endsWith(".spec.ts"))
    const workers = Math.max(1, Math.min(concurrency, specs.length))

    const results: { file: string; ok: boolean; out: string }[] = []
    const queue = [...specs]
    const runWorker = async () => {
      while (queue.length > 0) {
        const file = queue.shift()!
        try {
          const out = await this.test(file, source)
          results.push({ file, ok: true, out })
        } catch (e) {
          results.push({ file, ok: false, out: e instanceof Error ? e.message : String(e) })
        }
      }
    }
    await Promise.all(Array.from({ length: workers }, runWorker))

    results.sort((a, b) => specs.indexOf(a.file) - specs.indexOf(b.file))
    const summary = `${results.filter((r) => r.ok).length}/${results.length} passed (concurrency=${workers})`
    return results
      .map((r) => `${r.ok ? "✅" : "❌"} ${r.file}\n${r.out}`)
      .join("\n\n" + "=".repeat(80) + "\n\n") + "\n\n" + "=".repeat(80) + "\n" + summary
  }

  private appContainer(source: Directory): Container {
    return source.dockerBuild({
      dockerfile: "docker/Dockerfile",
    })
  }

  private postgresService(dbName: string): Service {
    return dag.container()
      .from("postgres:17-alpine")
      .withEnvVariable("POSTGRES_DB", dbName)
      .withEnvVariable("POSTGRES_USER", DB_USER)
      .withEnvVariable("POSTGRES_PASSWORD", DB_PASSWORD)
      .withExposedPort(5432)
      .asService()
  }

  private appService(source: Directory, postgres: Service, webhookMock: Service, oauthMock: Service, configToml: string, runId: string): Service {
    const startScript = `#!/bin/sh
set -e
cd /app
echo "Starting markpost backend..."
markpost -c /app/config.toml serve > /tmp/markpost.log 2>&1 &
MARKPOST_PID=$!
echo "Markpost PID: $MARKPOST_PID"

# Wait for backend to be ready on port 7330
echo "Waiting for backend to start on port 7330..."
for i in $(seq 1 30); do
  if curl -s http://127.0.0.1:7330/api/v1/health > /dev/null 2>&1; then
    echo "Backend is ready!"
    break
  fi
  if ! kill -0 $MARKPOST_PID 2>/dev/null; then
    echo "ERROR: Markpost process died!"
    cat /tmp/markpost.log
    exit 1
  fi
  sleep 1
done

# Check if backend is still running
if ! kill -0 $MARKPOST_PID 2>/dev/null; then
  echo "ERROR: Markpost process is not running!"
  cat /tmp/markpost.log
  exit 1
fi

echo "Starting Caddy..."
exec caddy run --config /etc/caddy/Caddyfile
`
    return this.appContainer(source)
      .withServiceBinding("postgres", postgres)
      .withServiceBinding("webhook-mock", webhookMock)
      .withServiceBinding("oauth-mock", oauthMock)
      .withNewFile("/app/config.toml", configToml)
      .withEnvVariable("RUN_ID", runId)
      .withExposedPort(2053)
      .withEntrypoint(["sh", "-c", startScript])
      .asService()
  }

  private webhookMockService(source: Directory, runId: string): Service {
    const buildAndRunScript = `#!/bin/sh
cd /app
npm install --include=dev
node_modules/.bin/tsc --esModuleInterop --target ES2020 --module commonjs --outDir . index.ts
exec node index.js
`
    return dag.container()
      .from("node:24-alpine")
      .withDirectory("/app", source.directory("e2e/mock-services/webhook-mock"))
      .withWorkdir("/app")
      .withExposedPort(3002)
      .withEnvVariable("RUN_ID", runId)
      .withEntrypoint(["sh", "-c", buildAndRunScript])
      .asService()
  }

  private oauthMockService(source: Directory, runId: string): Service {
    const buildAndRunScript = `#!/bin/sh
cd /app
npm install
exec node --import tsx index.ts
`
    return dag.container()
      .from("node:24-alpine")
      .withDirectory("/app", source.directory("e2e/mock-services/oauth-mock"))
      .withWorkdir("/app")
      .withEnvVariable("PORT", "3001")
      .withEnvVariable("MOCK_USER_ID", "12345")
      .withEnvVariable("MOCK_USERNAME", "testuser")
      .withEnvVariable("MOCK_EMAIL", "test@example.com")
      .withExposedPort(3001)
      .withEnvVariable("RUN_ID", runId)
      .withEntrypoint(["sh", "-c", buildAndRunScript])
      .asService()
  }

  private testRunner(source: Directory): Container {
    return dag.container()
      .from("mcr.microsoft.com/playwright:v1.53.0-noble")
      .withDirectory("/app/tests", source.directory("e2e/tests"))
      .withDirectory("/app/lib", source.directory("e2e/lib"))
      .withDirectory("/app/mock-services", source.directory("e2e/mock-services"))
      .withFile("/app/playwright.config.ts", source.file("e2e/playwright.config.ts"))
      .withFile("/app/package.json", source.file("e2e/package.json"))
      .withWorkdir("/app")
      .withExec(["npm", "install"])
  }
}
