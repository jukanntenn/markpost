# Authentication Specification

本文档定义 markpost 端到端的认证设计：JWT 双 token、Refresh 轮转与重用检测、OAuth（GitHub）同页重定向流程、密码登录、登出，以及前端的 token 存储与自动刷新。

## 一、JWT（HS256 双 token）

### 1.1 概览

认证基于无状态 JWT，access token 与 refresh token 分离：

| token 类型 | 用途 | 签名密钥 | 默认有效期 | 传输方式 |
|-----------|------|---------|-----------|---------|
| Access | API 请求鉴权 | `jwt.access_signing_key`（独立） | 24h | `Authorization: Bearer <token>` |
| Refresh | 换取新 access token | `jwt.refresh_signing_key`（独立） | 720h（30 天） | 请求体字段 |

两个 token 用**各自独立的 HMAC 密钥**签名，互不通用——access 密钥签的 token 无法通过 refresh 校验，反之亦然。

### 1.2 Claims 结构

```go
// Access token
type AccessClaims struct {
    UserID   int    `json:"user_id"`
    Email    string `json:"email"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims             // ExpiresAt / IssuedAt / NotBefore
}

// Refresh token
type RefreshClaims struct {
    UserID int    `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims             // ExpiresAt / IssuedAt
}
```

签发时三个时间戳全部设置：`ExpiresAt`（过期）、`IssuedAt`（签发）、`NotBefore`（生效，等于 IssuedAt）。

### 1.3 安全硬化（基于 golang-jwt/jwt v5 源码确认）

**锁定签名算法（防 alg:none / 算法混淆）**：

golang-jwt v5 的 `ParseWithClaims` 默认接受任意算法。攻击者可构造 `alg:none` 的 token 绕过签名校验。必须在解析时显式锁定允许的算法：

```go
func validateToken(tokenString string, key []byte) (jwt.Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, newClaims(), func(*jwt.Token) (any, error) {
        return key, nil
    }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
    // ...
}
```

> 依据：`jwt/parser_option.go:11-19` — `WithValidMethods` 的文档明确"heavily encouraged to prevent attacks such as algorithm confusion"。`jwt/parser.go:63-77` 校验 token 的 alg 必须在集合内，否则返回 `ErrTokenSignatureInvalid`。

**强制 exp claim**：

golang-jwt v5 默认在 `exp` 存在时校验，但**不要求**必须存在（无 exp 的 token 通过校验）。用 `WithExpirationRequired()` 强制：

```go
jwt.WithExpirationRequired()
```

> 依据：`jwt/parser_option.go:61-67` — "By default exp claim is optional."。防御性编程要求显式强制，即使我们签发时总带 exp。

**HMAC 密钥要求**：

- 配置校验 `access_signing_key` / `refresh_signing_key` **≥32 字节**（256 bit）
- golang-jwt **不**校验最小密钥长度（`jwt/hmac.go` 源码确认），须由应用层强制
- 密钥必须是 `crypto/rand` 生成的随机字节，不能用人类可读字符串
- `config.example.toml` 提供生成命令：

```toml
# [REQUIRED] 生成命令：openssl rand -base64 32
access_signing_key = "CHANGE_ME..."
refresh_signing_key = "CHANGE_ME..."
```

> 依据：`jwt/hmac.go:50-57` — "it is not advised to provide a []byte which was converted from a 'human readable' string... ideally be providing a []byte key which was produced from a cryptographically random source, e.g. crypto/rand."

### 1.4 Access token 黑名单

登出时将 access token 的 SHA-256 哈希存入 `token_blacklist` 表，TTL 设为 token 的剩余有效期。中间件 `AuthWithBlacklist` 每次请求查询黑名单：

```go
// 登出
tokenHash := utils.HashToken(accessToken)
expiresAt := time.Now().Add(ttl)  // ttl = token 剩余有效期
tokens.StoreBlacklistedToken(ctx, tokenHash, expiresAt)
```

短期 access token（24h）+ 黑名单的组合避免了维护服务端 session 的开销。token 过期后黑名单记录自然失效，由定期清理回收。

---

## 二、Refresh token 轮转与重用检测

### 2.1 一次性轮转

每次 refresh 都**吊销旧 refresh token + 签发新 token 对**（rotating refresh tokens）。Refresh 是一次性的，同一个 refresh token 不能用两次。

```
POST /auth/refresh { refresh_token }
  → 校验 refresh token（查库，未吊销 + 未过期）
  → 吊销该 refresh token（set revoked=true）
  → 签发新 access + refresh token 对
```

### 2.2 软标记吊销（revoked 字段）

`refresh_tokens` 表新增 `revoked` bool 字段（GORM AutoMigrate 自动添加，默认 false）：

| 字段 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `revoked` | bool | false | true 表示已吊销（用于重用检测） |

吊销操作从物理 `DELETE` 改为 `UPDATE SET revoked=true`。这样保留了吊销记录，使重用检测成为可能。

### 2.3 Token theft 重用检测

当一个**已被吊销**（`revoked=true`）的 refresh token 被再次提交时，判定为 token 被盗用，立即吊销该用户的**所有** refresh token：

```go
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) error {
    tokenHash := utils.HashToken(refreshToken)

    // 1. 查有效 token（revoked=false）
    record, err := s.tokens.GetRefreshToken(ctx, tokenHash)  // WHERE revoked = false
    if err == domain.ErrNotFound {
        // 2. 查是否是已吊销的 token（revoked=true）→ 重用 → theft
        revoked, err := s.tokens.IsRefreshTokenRevoked(ctx, tokenHash)  // WHERE revoked = true
        if revoked {
            // token theft：吊销该用户所有 refresh token
            s.tokens.RevokeAllByUserID(ctx, record.UserID)
            return service.New(service.ErrInvalidToken, "refresh token reuse detected")
        }
        return service.New(service.ErrInvalidToken, "invalid refresh token")
    }
    // 3. 正常轮转：吊销旧 + 签发新
    // ...
}
```

**为什么需要软标记**：物理删除（`DELETE`）无法区分"token 从来不存在" vs "token 存在过但已被吊销"。软标记保留了吊销记录，使两者可区分——前者是正常无效，后者是 theft 信号。

### 2.4 查询规则

- `GetRefreshToken`：`WHERE token_hash = ? AND revoked = false`（只返回有效 token）
- `IsRefreshTokenRevoked`：`WHERE token_hash = ? AND revoked = true`（重用检测）
- `RevokeAllByUserID`：`UPDATE refresh_tokens SET revoked = true WHERE user_id = ? AND revoked = false`（thief 后全吊销）
- 过期 + revoked 的行仍可正常 prune 清理

---

## 三、OAuth（GitHub）— 同页重定向

### 3.1 交互模式：同页重定向（放弃弹窗）

markpost 采用**同页重定向**（模式 B），而非弹窗模式。这是基于消除本质缺陷的决策：

**弹窗模式的本质缺陷**（无论用轮询 localStorage 还是 postMessage）：
- 弹窗可能被浏览器拦截
- 用户手动关闭弹窗时，主窗口无法区分"成功关闭"还是"失败关闭"
- 跨窗口通信（postMessage）需要处理 origin 校验、弹窗引用丢失等边缘情况
- 移动端弹窗体验差

**同页重定向消除了这些问题**：
- 没有第二个窗口，无需跨窗口通信
- 所有状态在同一页面会话内流转
- 失败时 callback 页面在同一上下文，直接处理 error 分支
- 移动端友好（标准浏览器导航）

代价是用户点 GitHub 登录后，浏览器会整页跳转到 GitHub 授权页，授权后跳回。这一瞬的"离开页面"是标准 OAuth 体验，所有主流应用（Google、GitHub、Auth0、NextAuth）都这么做。对 markpost（登录是低频操作）无感。

### 3.2 完整流程

```
① 用户点击 "GitHub 登录"
   前端: GET /api/v1/oauth/url
   ──────────────────────────────────────────────
② 后端 /oauth/url:
   - 生成 state: crypto/rand 20 字节 → base64url
   - 生成 verifier: oauth2.GenerateVerifier()
   - 存 ristretto: key=state, value={verifier, createdAt}, TTL=10min
   - 构造授权 URL: oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
   - 返回 { url, state }
   ──────────────────────────────────────────────
③ 前端拿到 { url, state }:
   - 存 expectedState 到 sessionStorage（key="oauth_state"）
   - location.href = url（整页跳转 GitHub）
   ──────────────────────────────────────────────
④ GitHub 授权页: 用户登录 / 授权
   ──────────────────────────────────────────────
⑤ GitHub 重定向回 redirect_uri:
   成功: /auth/callback?code=xxx&state=yyy
   失败: /auth/callback?error=access_denied
   ──────────────────────────────────────────────
⑥ 前端 /auth/callback 页面（静态）:
   - 从 URL query 解析 code, state（或 error）
   - 失败分支（有 error）: 显示错误 → router.replace('/login')
   - 成功分支:
     a. 二次校验: 回调的 state === sessionStorage 的 expectedState
     b. 清除 sessionStorage 的 oauth_state
     c. POST /api/v1/oauth/login { code, state }
   ──────────────────────────────────────────────
⑦ 后端 /oauth/login:
   - 校验 state: 从 ristretto 查 state → 必须存在且未过期
   - 取出 verifier
   - 立即从 ristretto 删除该 state（一次性消费，防重放）
   - oauth2.Exchange(ctx, code, oauth2.VerifierOption(verifier))
   - 获取 GitHub 用户信息（/user + /user/emails）
   - GetOrCreateFromGitHub
   - completeLogin: 签发 token 对
   - 返回 { user, token, refresh_token, expires_in }
   ──────────────────────────────────────────────
⑧ 前端 callback:
   - setAuth(token, user, refresh_token) → 存 localStorage
   - router.replace('/dashboard')
```

### 3.3 State 校验（CSRF 防护）

**双层校验**：

| 层 | 存储 | 职责 |
|----|------|------|
| 后端（主防线） | ristretto 缓存 | 生成 state 时存，`/oauth/login` 时校验匹配 + 一次性消费 |
| 前端（二次校验） | sessionStorage | `/oauth/url` 返回的 state 存 sessionStorage，callback 时比对 |

后端是主防线（state 不匹配 → 401）。前端二次校验提前拦截，减少无效请求。

> **oauth2 库不做 state 校验**：`golang.org/x/oauth2` 的 `AuthCodeURL` 只是把 state 放进 URL，`Exchange` 不校验 state。文档明确："be sure to validate `http.Request.FormValue("state")`... this is the application's responsibility."（`oauth2.go:217-218`）。state 校验完全由应用层负责。

### 3.4 PKCE（Proof Key for Code Exchange）

在 state 基础上加 PKCE 双保险：

| 组件 | 位置 | 说明 |
|------|------|------|
| `verifier` | 随 state 同存 ristretto（不进 URL） | `oauth2.GenerateVerifier()`，32 字节随机 |
| `challenge` | 授权 URL（`code_challenge` 参数） | `S256ChallengeOption(verifier)` 计算 SHA256(verifier) |
| Exchange | `VerifierOption(verifier)` | 后端 Exchange 时传 verifier，GitHub 校验 SHA256(verifier) === challenge |

> 依据：`golang.org/x/oauth2/pkce.go` — `GenerateVerifier()`（`pkce.go:27-38`）、`S256ChallengeOption`（`pkce.go:57-62`）、`VerifierOption`（`pkce.go:42-44`）。`oauth2.go:153-158` 建议用 PKCE 做 CSRF 防护。

**verifier 随 state 同存**：state 和 verifier 绑定，存为同一个 ristretto 条目（`key=state, value={verifier, createdAt}`）。`/oauth/login` 时用 state 查出 verifier，一次查询拿到两者。

### 3.5 State / verifier 存储

存储介质：**ristretto 缓存**（项目已用，render cache 同款），TTL 10 分钟。

理由：state/verifier 是一次性、短生命周期的数据（生成后几分钟内被消费），内存缓存最合适，不增新依赖，不入 DB。

### 3.6 redirect_uri 与 callback 路由

GitHub OAuth App 的 `redirect_uri` 注册为：`https://<your-domain>/auth/callback`

前端路由 `/auth/callback`（放在 `(auth)` 路由组，用 PublicRoute 守卫）。

**不带 provider**：所有 provider（GitHub/Google/微信）共用同一个 `/auth/callback`。provider 信息编码在 state 里（后端存 `state → {provider, verifier}`），前端回调逻辑统一（取 code+state → POST → 存 token → 跳转）。扩展新 provider 是纯后端改动。

### 3.7 错误处理矩阵

每个失败路径都有明确的用户可见行为：

| 失败场景 | 检测点 | HTTP/状态 | 用户可见行为 |
|---------|--------|----------|-------------|
| 用户拒绝授权 | GitHub 重定向带 `?error=access_denied` | callback 前端 | 提示"授权已取消" → `/login` |
| state 前端不匹配 | callback state ≠ sessionStorage | callback 前端 | 提示"登录状态异常，请重试" → `/login` |
| state 后端不匹配/过期 | ristretto 查不到 state | `/oauth/login` 401 `invalid_state` | 提示"登录超时，请重试" → `/login` |
| state 重复使用（重放） | ristretto 已删除（一次性消费） | `/oauth/login` 401 `invalid_state` | 同上 |
| PKCE 校验失败 | Exchange 时 verifier 不匹配 | `/oauth/login` 401 `oauth_exchange_failed` | 提示"授权验证失败" → `/login` |
| GitHub Exchange 失败 | token endpoint 拒绝 | `/oauth/login` 401 `oauth_exchange_failed` | 同上 |
| 获取 GitHub 用户信息失败 | API 调用失败 | `/oauth/login` 502 `github_user_fetch_failed` | 提示"无法获取 GitHub 账户信息" → `/login` |
| code 缺失/格式错 | callback query 无 code | callback 前端 | 提示"授权回调无效" → `/login` |
| 用户关 GitHub 页面/返回 | 无回调发生 | 前端无感知 | 用户回到登录页需重新点击（无"卡在等待"状态） |
| 网络断开 | POST /oauth/login fetch 失败 | callback 前端 catch | 显示"网络错误，请重试"，保留在 callback 页 |

### 3.8 OAuth 错误码

定义在 `internal/service/auth/errors.go`（遵循 error-handling.md 的"域专属码分文件"原则）：

| ErrCode | Value | HTTP | 场景 |
|---------|-------|------|------|
| `ErrMissingState` | `missing_state` | 400 | `/oauth/login` 请求缺 state 参数 |
| `ErrMissingCode` | `missing_code` | 400 | `/oauth/login` 请求缺 code 参数 |
| `ErrInvalidState` | `invalid_state` | 401 | state 不匹配 / 过期 / 重放 |
| `ErrOAuthExchangeFailed` | `oauth_exchange_failed` | 401 | PKCE 校验失败 或 GitHub Exchange 失败 |
| `ErrGitHubUserFetch` | `github_user_fetch_failed` | 502 | 获取 GitHub 用户信息失败（上游故障，用 502 Bad Gateway 而非笼统 500） |

---

## 四、密码登录

### 4.1 哈希

使用 `golang.org/x/crypto/bcrypt`：

- **Cost**：`bcrypt.DefaultCost`（10）。有效范围 4–31，DefaultCost 是唯一推荐值。
- **盐**：`GenerateFromPassword` 内部用 `crypto/rand` 生成 16 字节随机盐，调用方无需提供。
- **校验**：`CompareHashAndPassword` 用 `crypto/subtle.ConstantTimeCompare` 做常量时间比较，防时序攻击。不匹配返回 `ErrMismatchedHashAndPassword`。

> 依据：`crypto/bcrypt/bcrypt.go:95-98`（GenerateFromPassword）、`bcrypt.go:153-154`（crypto/rand 盐）、`bcrypt.go:120`（常量时间比较）、`bcrypt.go:29`（ErrMismatchedHashAndPassword）。

### 4.2 密码长度策略

| 约束 | 值 | 理由 |
|------|-----|------|
| 最小长度 | 8 字符 | NIST 800-63B 建议：长度比复杂度更重要 |
| 最大长度 | 72 字符 | bcrypt 算法限制（见下） |
| 复杂度 | **不强制** | NIST 800-63B 不推荐强制大小写+数字+符号（促使用户用可预测替换如 `P@ssw0rd!`） |

### 4.3 72 字节上限预检

bcrypt 算法只处理 ≤72 字节的密码。`GenerateFromPassword` 对超长密码返回 `ErrPasswordTooLong`（拒绝，不截断）。

> 依据：`bcrypt/bcrypt.go:96-98` — `if len(password) > 72 { return nil, ErrPasswordTooLong }`。

在 `SetPassword` / `ChangePassword` 中**预检**长度，返回友好错误（而非 bcrypt 的原始 error）：

```go
if utf8.RuneCountInString(password) > 72 {
    return service.New(service.ErrValidation, "password must not exceed 72 characters")
}
```

注意用 `utf8.RuneCountInString`（按字符计数）而非 `len`（按字节），因为中文字符是多字节——72 个中文字符的字节数远超 72，但语义上是 72 个字符。bcrypt 的 72 字节限制是字节级的，所以实际校验要同时满足"字符数合理"且"字节数 ≤72"。

---

## 五、登出

登出同时处理两种 token：

| token | 登出操作 |
|-------|---------|
| Access token | SHA-256 哈希存入 `token_blacklist`（TTL = 剩余有效期），中间件 `AuthWithBlacklist` 后续拒绝 |
| Refresh token | `UPDATE refresh_tokens SET revoked=true WHERE user_id = ?`（吊销该用户的所有 refresh token） |

登出吊销 refresh token 防止攻击者在 access token 过期后用残留的 refresh token 重新获取访问权限。

---

## 六、前端 Token 存储与刷新

### 6.1 存储

| 项 | 设计 |
|----|------|
| 存储位置 | `localStorage`（key = `markpost_auth`） |
| 存储内容 | `{ token, refreshToken, user, _hasHydrated }` |
| 状态管理 | Zustand + `persist` 中间件，`partialize` 只持久化 token/refreshToken/user |

### 6.2 XSS 风险与缓解

localStorage 对所有同源 JS 可见，任何 XSS（含第三方库漏洞）都能窃取 token。

**为什么接受 localStorage**：前端重构为纯静态客户端（无服务端运行时），无法使用 HttpOnly cookie（没有服务端 Set-Cookie）。access + refresh 都存 localStorage 是纯静态前端唯一可行的方案。

**缓解措施**：
- CSP（Content-Security-Policy）限制脚本来源
- 所有用户输入经过 bluemonday 消毒（文章渲染）+ 输出转义（模板）
- 依赖项定期审计

### 6.3 自动刷新（401 拦截）

API client 拦截 401 响应，自动尝试 refresh：

```typescript
// 伪代码（详见 src/lib/api/base.ts）
async function handleTokenRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;  // 单飞：并发 401 共享一个 refresh
  refreshPromise = refreshAccessToken().finally(() => { refreshPromise = null; });
  return refreshPromise;
}

// request 函数内：
if (response.status === 401 && !skipAuthRefresh) {
  const refreshed = await handleTokenRefresh();
  if (!refreshed) throw new Error("Session expired");
  return retry();  // 用新 token 重试原请求
}
```

**单飞（single-flight）**：多个并发请求同时 401 时，只发一次 refresh，所有请求共享结果（`refreshPromise` 去重）。避免 refresh token 被消耗多次（一次性轮转会拒绝第二次）。

刷新失败 → `logout()`（清空 localStorage）→ 后续请求无 token → 路由守卫重定向到 `/login`。

### 6.4 水合处理

Zustand persist 从 localStorage 恢复是异步的。用 `_hasHydrated` 标志防止水合前用默认空状态（`token=null`）误判"未认证"导致闪烁跳转：

```typescript
onRehydrateStorage: () => (state) => {
  state?.setHasHydrated(true);
}
```

路由守卫在水合完成前渲染 PageSpinner，水合后根据真实认证状态决定渲染/重定向。

### 6.5 Accept-Language 头

API client 在每个请求的 header 中携带 `Accept-Language: <当前 locale>`，后端据此返回对应语言的错误消息。详见 [frontend/i18n.md](./frontend/i18n.md)。

---

## 七、OAuth Callback 页面职责

`/auth/callback` 页面（`(auth)` 路由组，PublicRoute 守卫）的处理逻辑：

```typescript
// 伪代码
function AuthCallbackPage() {
  const searchParams = useSearchParams();
  const setAuth = useAuthStore((s) => s.setAuth);

  useEffect(() => {
    const code = searchParams.get("code");
    const state = searchParams.get("state");
    const error = searchParams.get("error");

    // 1. 失败分支（GitHub 返回 error）
    if (error) {
      router.replace("/login");
      return;
    }

    // 2. 参数校验
    if (!code || !state) {
      router.replace("/login");
      return;
    }

    // 3. 前端二次校验 state
    const expectedState = sessionStorage.getItem("oauth_state");
    if (state !== expectedState) {
      router.replace("/login");
      return;
    }
    sessionStorage.removeItem("oauth_state");

    // 4. POST 后端
    authApi.loginWithGitHub(code, state)
      .then((data) => {
        setAuth(data.token, data.user, data.refresh_token);
        router.replace("/dashboard");
      })
      .catch(() => {
        router.replace("/login");
      });
  }, []);
}
```

所有失败路径都 `router.replace('/login')`（不留在 callback 页），成功路径 `router.replace('/dashboard')`。

---

## 参考

- [error-handling.md](./backend/error-handling.md) — ErrCode struct、错误响应格式、域专属码分文件
- [frontend/routes.md](./frontend/routes.md) — 路由守卫架构、安全边界声明
- [api-design.md](./api-design.md) — `/oauth/*`、`/auth/*` 端点设计
