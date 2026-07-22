# E2E 测试手册

> 面向测试人员的快速入门指南，涵盖架构、执行、调试和常见问题。

---

## 1. 架构概览

```
┌──────────────┐     ┌──────────────────────┐     ┌──────────────┐
│   Playwright │────▶│   App Container      │────▶│  PostgreSQL  │
│   测试运行器  │     │  Caddy (HTTPS:2053)  │     │   (5432)     │
│  (宿主机)    │     │  + Go 后端 (7330)    │     │              │
└──────────────┘     │  + 前端静态文件       │     └──────────────┘
                     └──────────────────────┘
                            │
                            ├─────────────────┐
                            ▼                 ▼
                     ┌──────────────┐  ┌──────────────┐
                     │ Webhook Mock │  │  OAuth Mock  │
                     │  (3002)      │  │  (3001)      │
                     └──────────────┘  └──────────────┘
```

### 核心组件

| 组件 | 作用 | 端口 |
|------|------|------|
| **App Container** | Caddy 反向代理 + Go 后端 + 前端静态文件 | 2053 (HTTPS) |
| **PostgreSQL** | 测试数据库，每个测试文件独立数据库 | 5432 (内部) |
| **Webhook Mock** | 模拟飞书 Webhook 接收端 | 3002 (内部) |
| **OAuth Mock** | 模拟 GitHub OAuth 全流程（授权、令牌交换、用户信息） | 3001 (内部) |
| **Playwright** | 浏览器自动化测试运行器（宿主机） | - |

### 关键设计决策

1. **唯一入口 HTTPS 2053**：与生产环境一致，Caddy 使用 `tls internal` 自签证书，不暴露 HTTP 8080
2. **数据隔离**：每个测试文件使用独立数据库名 `markpost_{runId}`
3. **测试数据清理**：每个测试前后通过 API 清理数据，确保测试独立性
4. **统一 Dockerfile/Caddyfile**：轻量反馈测试、生产环境、Dagger 全量测试共用同一套 `docker/Dockerfile` 和 `docker/Caddyfile`

---

## 2. 快速开始

### 前置条件

- Docker 已安装并运行
- 项目根目录下执行命令

### 方式一：Docker Compose 快速反馈（推荐开发时使用）

```bash
# 1. 启动服务（首次会构建镜像）
docker compose -f e2e/docker-compose.yml up -d --build

# 2. 等待服务就绪（约 30 秒）
curl -k https://localhost:2053/api/v1/health
# 应返回: {"status":"ok"}

# 3. 运行所有测试（从宿主机执行）
cd e2e && pnpm test

# 4. 运行单个测试文件
cd e2e && npx playwright test tests/login.spec.ts --reporter=list

# 5. 停止服务并清理数据
docker compose -f e2e/docker-compose.yml down -v
```

### 方式二：Dagger（CI 环境使用）

```bash
# 运行所有测试（自动构建镜像、启动服务、隔离数据库）
cd e2e
dagger call all --source=..

# 运行单个测试文件
dagger call test --source=.. --test-file=login.spec.ts
```

---

## 3. 测试覆盖范围

### 测试文件清单

| 测试文件 | 覆盖功能 |
|---------|---------|
| `login.spec.ts` | 登录表单、验证、错误提示、键盘提交、重定向 |
| `dashboard.spec.ts` | Post Key 显示/隐藏/复制、用户菜单、登出 |
| `dashboard-create-post.spec.ts` | 快速创建帖子、表单验证 |
| `posts.spec.ts` | 帖子列表页、未认证重定向 |
| `admin.spec.ts` | 管理页权限、导航链接 |
| `admin-users.spec.ts` | 用户列表、管理员显示 |
| `admin-posts.spec.ts` | 帖子管理、搜索功能 |
| `admin-channels.spec.ts` | 渠道列表、创建渠道 |
| `admin-delivery-history.spec.ts` | 投递历史空状态 |
| `settings.spec.ts` | 设置页渲染、语言切换、密码验证 |
| `settings-change-password.spec.ts` | 修改密码并验证登录 |
| `settings-delivery-channel.spec.ts` | 渠道 CRUD、启用/禁用切换 |
| `delivery-history.spec.ts` | 投递历史区域显示 |
| `oauth-callback.spec.ts` | OAuth 全流程：成功登录、缺少参数、错误参数、无效 state、令牌交换失败、用户信息获取失败、state 一次性消费 |
| `feishu-webhook.spec.ts` | Webhook 触发与负载验证 |

---

## 4. 项目结构

```
e2e/
├── docker-compose.yml             # 快速反馈 Docker Compose 配置
├── tests/                         # 测试文件
├── lib/
│   ├── fixtures.ts                # Playwright fixtures（认证、页面对象）
│   ├── helpers.ts                 # API 辅助函数（登录、创建数据、清理）
│   └── pages/                     # 页面对象模型
├── mock-services/
│   ├── oauth-mock/                # GitHub OAuth Mock（基于 oauth2-mock-server）
│   │   ├── index.ts
│   │   ├── Dockerfile
│   │   └── package.json
│   └── webhook-mock/              # 飞书 Webhook Mock 服务
│       ├── index.ts
│       ├── Dockerfile
│       └── package.json
├── src/                           # Dagger 模块
│   └── src/index.ts
├── playwright.config.ts           # Playwright 配置
├── package.json
└── HANDBOOK.md                    # 本手册
```

---

## 5. 页面对象

所有页面对象位于 `e2e/lib/pages/`，封装了页面交互逻辑。

---

## 6. 测试数据管理

### 数据清理

每个测试的 `beforeEach` 和 `afterEach` 中调用 `cleanupTestData()`：

```typescript
import { test, expect, cleanupTestData } from "../lib/fixtures";

test.beforeEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});
```

`cleanupTestData` 函数会：
1. 删除所有帖子
2. 删除所有投递渠道
3. 清空 Webhook Mock 接收记录
4. 清空 OAuth Mock 请求记录

---

## 7. 环境变量

| 变量 | 默认值 | 说明 |
|------|-------|------|
| `BASE_URL` | `https://localhost:2053` | 前端访问地址 |
| `BACKEND_URL` | `https://localhost:2053` | 后端 API 地址 |
| `ADMIN_USERNAME` | `markpost` | 管理员用户名 |
| `ADMIN_PASSWORD` | `markpost` | 管理员密码 |
| `WEBHOOK_MOCK_URL` | `http://localhost:3002` | Webhook Mock 地址 |
| `OAUTH_MOCK_URL` | `http://localhost:3001` | OAuth Mock 地址 |
| `NODE_TLS_REJECT_UNAUTHORIZED` | - | 设为 `0` 跳过 TLS 验证 |

---

## 8. 常见问题与解决方案

### Q1: TLS 握手失败 `SSL routines:ssl3_read_bytes:tlsv1 alert internal error`

**原因**：Caddy 自签证书的 SNI 与请求主机名不匹配。

**解决**：
- Docker Compose：设置 `NODE_TLS_REJECT_UNAUTHORIZED=0`
- Dagger：已在模块中设置该环境变量

### Q2: CORS 配置导致后端 panic

**原因**：Viper 无法从环境变量解析 JSON 数组格式的 CORS origins。

**解决**：使用 `config.toml` 文件配置 CORS，不要通过环境变量传递数组值。

### Q3: 测试间数据污染

**原因**：前一个测试创建的数据影响后续测试。

**解决**：
- 每个测试文件的 `beforeEach` 中调用 `cleanupTestData()`
- 密码修改测试需要在修改前和重置后都清理数据
- API 响应格式为 `data.items`（不是 `data.channels`）

### Q4: Webhook 测试收不到回调

**原因**：Webhook Mock 服务未绑定到 App 容器。

**解决**：Dagger 模块中 `appService` 需要 `withServiceBinding("webhook-mock", webhookMock)`。

### Q5: OAuth 测试失败

**原因**：OAuth Mock 服务未启动或后端未配置指向 Mock 服务。

**解决**：
- 确保 `e2e/docker-compose.yml` 中 `oauth-mock` 服务正常运行
- 确保后端环境变量包含 `MARKPOST_OAUTH__GITHUB__AUTH_URL`、`TOKEN_URL`、`USER_URL`

### Q6: 首次运行 Dagger 很慢

**原因**：首次需要下载所有依赖和构建镜像。

**解决**：开发时使用 Docker Compose 快速反馈，验证通过后再用 Dagger。

---

## 9. 调试技巧

### 查看测试失败截图

测试失败时会自动生成截图到 `e2e/test-results/` 目录。

### 查看后端日志

```bash
# Docker Compose
docker compose -f e2e/docker-compose.yml logs app --tail=50

# 查看 GIN 请求日志
docker compose -f e2e/docker-compose.yml exec app cat /tmp/markpost.log
```

### 测试单个用例

```bash
cd e2e && npx playwright test tests/login.spec.ts -g "logs in with valid" --reporter=list
```

---

## 10. CI 集成

### GitHub Actions 配置

```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install Dagger
        run: curl -fsSL https://dl.dagger.io/dagger/install.sh | sh
      - name: Run E2E tests
        run: cd e2e && dagger call all --source=..
```

---

## 11. 注意事项

1. **唯一入口 HTTPS 2053**：不暴露 HTTP 8080，与生产环境一致
2. **Docker Compose 不挂载数据卷**：测试数据是临时的，容器停止后自动清理
3. **测试必须串行执行**：`workers: 1` 确保数据隔离
4. **每个测试都要清理数据**：避免测试间数据污染
5. **统一配置**：轻量反馈测试、生产环境、Dagger 全量测试共用 `docker/Dockerfile` 和 `docker/Caddyfile`，不维护单独的 E2E 配置

---

*手册版本: 2026-07-16*
