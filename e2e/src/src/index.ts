import { dag, Container, Directory, object, func, ReturnType } from "@dagger.io/dagger"

const DB_NAME = "markpost"
const DB_USER = "markpost"
const DB_PASSWORD = "markpost"
const ADMIN_USERNAME = "markpost"
const ADMIN_PASSWORD = "markpost"
const ACCESS_SIGNING_KEY = "e2e-access-key"
const REFRESH_SIGNING_KEY = "e2e-refresh-key"

@object()
export class MarkpostE2E {
  @func()
  async test(testFile: string, source: Directory): Promise<string> {
    const e2eDir = source.directory("e2e")
    const runId = testFile + "-" + Date.now()

    const backendImg = this.buildBackend(source.directory("backend"))
    const frontendImg = this.buildFrontend(source.directory("frontend"))
    const postgres = this.postgresService(runId)

    const backendSvc = this.backendService(backendImg, source.directory("backend"), postgres, runId)
    const frontendSvc = this.frontendService(frontendImg, backendSvc, runId)

    const runner = this.playwrightRunner(e2eDir)
      .withServiceBinding("frontend", frontendSvc)
      .withServiceBinding("backend", backendSvc)
      .withEnvVariable("FRONTEND_URL", "http://frontend:3000")
      .withEnvVariable("BACKEND_URL", "http://backend:7330")
      .withEnvVariable("ADMIN_USERNAME", ADMIN_USERNAME)
      .withEnvVariable("ADMIN_PASSWORD", ADMIN_PASSWORD)
      .withWorkdir("/app")
      .withExec(["npx", "playwright", "test", "--config=playwright.config.ts", testFile], { expect: ReturnType.Any })

    const exitCode = await runner.exitCode()
    const stdout = await runner.stdout()
    if (exitCode !== 0) {
      throw new Error(stdout)
    }
    return stdout
  }

  @func()
  async all(source: Directory): Promise<string> {
    const testFiles = await source.directory("e2e/tests").entries()
    const specs = testFiles.filter((f: string) => f.endsWith(".spec.ts"))
    const results: string[] = []
    for (const file of specs) {
      try {
        const out = await this.test(file, source)
        results.push(`✅ ${file}\n${out}`)
      } catch (e) {
        results.push(`❌ ${file}\n${e instanceof Error ? e.message : String(e)}`)
      }
    }
    return results.join("\n\n" + "=".repeat(80) + "\n\n")
  }

  private buildBackend(source: Directory): Container {
    return dag.container()
      .from("golang:1.26-alpine")
      .withExec(["apk", "add", "--no-cache", "gcc", "musl-dev", "sqlite-dev"])
      .withMountedCache("/go/pkg/mod", dag.cacheVolume("go-mod-e2e-v4"))
      .withMountedCache("/root/.cache/go-build", dag.cacheVolume("go-build-e2e-v4"))
      .withDirectory("/app", source)
      .withWorkdir("/app")
      .withExec(["go", "mod", "download"])
      .withEnvVariable("CGO_ENABLED", "1")
      .withExec(["go", "build", "-o", "markpost", "./cmd/server"])
  }

  private buildFrontend(source: Directory): Container {
    const builder = dag.container()
      .from("node:24-alpine")
      .withExec(["npm", "install", "-g", "pnpm@11"])
      .withEnvVariable("CI", "true")
      .withMountedCache("/root/.local/share/pnpm", dag.cacheVolume("pnpm-store-e2e"))
      .withDirectory("/app", source)
      .withWorkdir("/app")
      .withExec(["pnpm", "install", "--frozen-lockfile"])
      .withExec(["pnpm", "build"])

    const standalone = builder.directory("/app/.next/standalone")
    const staticFiles = builder.directory("/app/.next/static")
    const publicFiles = builder.directory("/app/public")

    return dag.container()
      .from("node:24-alpine")
      .withDirectory("/app", standalone)
      .withDirectory("/app/.next/static", staticFiles)
      .withDirectory("/app/public", publicFiles)
      .withWorkdir("/app")
      .withEnvVariable("HOSTNAME", "0.0.0.0")
      .withEnvVariable("PORT", "3000")
      .withEnvVariable("NODE_ENV", "production")
      .withExposedPort(3000)
      .withEntrypoint(["node", "server.js"])
  }

  private postgresService(runId: string): Container {
    return dag.container()
      .from("postgres:17-alpine")
      .withEnvVariable("POSTGRES_DB", DB_NAME)
      .withEnvVariable("POSTGRES_USER", DB_USER)
      .withEnvVariable("POSTGRES_PASSWORD", DB_PASSWORD)
      .withEnvVariable("RUN_ID", runId)
      .withExposedPort(5432)
      .asService({ useEntrypoint: true })
  }

  private backendService(backendBuilder: Container, backendSource: Directory, postgres: Container, runId: string): Container {
    return backendBuilder
      .withDirectory("/app/templates", backendSource.directory("templates"))
      .withDirectory("/app/locales", backendSource.directory("locales"))
      .withEnvVariable("MARKPOST_SERVER__HOST", "0.0.0.0")
      .withEnvVariable("MARKPOST_SERVER__PORT", "7330")
      .withEnvVariable("MARKPOST_DEBUG", "true")
      .withEnvVariable("MARKPOST_DB__DRIVER", "postgresql")
      .withEnvVariable("MARKPOST_DB__DSN", `postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable`)
      .withEnvVariable("MARKPOST_JWT__ACCESS_SIGNING_KEY", ACCESS_SIGNING_KEY)
      .withEnvVariable("MARKPOST_JWT__REFRESH_SIGNING_KEY", REFRESH_SIGNING_KEY)
      .withEnvVariable("MARKPOST_ADMIN__INITIAL_USERNAME", ADMIN_USERNAME)
      .withEnvVariable("MARKPOST_ADMIN__INITIAL_PASSWORD", ADMIN_PASSWORD)
      .withEnvVariable("RUN_ID", runId)
      .withServiceBinding("postgres", postgres)
      .withExposedPort(7330)
      .withDefaultArgs(["/app/markpost"])
      .asService({ useEntrypoint: true })
  }

  private frontendService(frontendImg: Container, backendSvc: Container, runId: string): Container {
    return frontendImg
      .withEnvVariable("RUN_ID", runId)
      .withEnvVariable("BACKEND_URL", "http://backend:7330")
      .withServiceBinding("backend", backendSvc)
      .asService({ useEntrypoint: true })
  }

  private playwrightRunner(e2eDir: Directory): Container {
    return dag.container()
      .from("mcr.microsoft.com/playwright:v1.53.0-noble")
      .withDirectory("/app/tests", e2eDir.directory("tests"))
      .withDirectory("/app/lib", e2eDir.directory("lib"))
      .withFile("/app/playwright.config.ts", e2eDir.file("playwright.config.ts"))
      .withFile("/app/package.json", e2eDir.file("package.json"))
      .withWorkdir("/app")
      .withExec(["npm", "install"])
  }
}
