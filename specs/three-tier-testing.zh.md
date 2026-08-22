# 三层测试策略

[English](three-tier-testing.md) | 中文

<a id="tier-1-unit-tests"></a>

## Tier 1：单元测试

<a id="backend"></a>

### 后端

- 所有测试使用真实 PostgreSQL testcontainer（testcontainers-go）
- 仅在难以控制的场景（如注入 DB 错误）使用接口级 mock
- 外部 HTTP 服务（邮件、云 API）用 `net/http/httptest`
- 把现有 mock-repo service 测试按域分批重写为真实 DB
- 边界场景（约束、分页）的 repository 测试
- 使用现有 `testutil` engine 的 handler 测试

<a id="frontend"></a>

### 前端

- 加强现有单元/组件测试
- 确保 MSW handler 覆盖完整 API 契约
- 补齐缺失的组件测试

<a id="tier-2-integration-tests"></a>

## Tier 2：集成测试

- Go 测试二进制统筹：真实后端（PostgreSQL）+ `httptest.Server`（外部服务）+ Playwright
- 前端 dev server 作为子进程启动
- 把现有 E2E 测试从 mock API 路由迁移到真实后端
- 在 PR 上运行，作为合入闸门

<a id="tier-3-agent-driven-tests"></a>

## Tier 3：Agent 驱动测试

<a id="scn-format"></a>

### SCN 格式

混合式 —— 准备/清理用 API 步骤，用户可见行为用浏览器步骤。

```
SCN-001: Admin publishes a post
- Backend healthy (GET /health -> 200)
- Admin authenticated (API token, role=ADMIN)

1. Create test post via API.
   POST /api/v1/posts {title: "Test", body: "Content"} -> 201
2. Navigate to posts page.
   BROWSER: goto /posts
3. Verify post appears in list.
   BROWSER: see "Test" in post list
4. Click publish.
   BROWSER: click "Publish" on post "Test"
5. Verify status changed.
   BROWSER: see "Published" badge
6. Verify via API.
   GET /api/v1/posts/1 -> 200, status=published
```

<a id="execution"></a>

### 执行

- 主 Claude agent 通过 `/playwright-cli` 对 `devops/dev.py` 环境执行 SCN
- 没有 subagent 流水线 —— 人在环中的编排
- 人审阅 SCN，调用 agent 执行

<a id="authentication"></a>

### 认证

- Docker 环境中预置测试用户与令牌
- API 步骤和大多数浏览器流程使用注入的令牌
- 仅专门测试认证行为的 SCN 走 UI 登录

<a id="mocking"></a>

### Mock

- 几乎没有 mock —— 所有服务都是真实的
- 仅 mock：OAuth2 和真正难以运行的第三方服务

<a id="ci"></a>

### CI

- 按需 + 每晚运行，不是合入闸门
- 波动（flaky）被接受，由人分诊失败

<a id="artifact-locations"></a>

## 产物位置

| 产物              | 位置                                   |
| ----------------- | -------------------------------------- |
| 领域聚合          | `specs/aggregates/`                    |
| 结构化场景（SCN） | `tests/e2e/scenarios/`                 |
| Agent 工具        | `.claude/skills/`、`.claude/commands/` |

<a id="phasing"></a>

## 阶段划分

1. **Phase 1a** —— Tier 1 后端：按域把现有 mock 测试重写为真实 DB，构建共享测试助手
2. **Phase 1b** —— Tier 1 前端：加强现有单元/组件测试
3. **Phase 2** —— Tier 2：Go 集成测试骨架，把 E2E 测试迁移到真实后端
4. **Phase 3** —— Tier 3：SCN 格式、首批场景、agent 对 dev 环境执行
