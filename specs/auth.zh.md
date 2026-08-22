# 认证规范

[English](auth.md) | 中文

本文档定义 markpost 端到端的认证设计：JWT 双令牌、刷新令牌轮转与重用检测、OAuth（GitHub）同页重定向流程、密码登录、登出，以及前端的令牌存储与自动刷新。

<a id="1-jwt-hs256-dual-tokens"></a>

## 一、JWT（HS256 双令牌）

<a id="11-overview"></a>

### 1.1 概览

认证基于无状态 JWT，访问令牌与刷新令牌分离：

| 令牌类型 | 用途           | 签名密钥                          | 默认有效期    | 传输方式                        |
| -------- | -------------- | --------------------------------- | ------------- | ------------------------------- |
| 访问令牌 | API 请求鉴权   | `jwt.access_signing_key`（独立）  | 24h           | `Authorization: Bearer <token>` |
| 刷新令牌 | 换取新访问令牌 | `jwt.refresh_signing_key`（独立） | 720h（30 天） | 请求体字段                      |

两个令牌用**各自独立的 HMAC 密钥**签名，互不通用——访问密钥签的令牌无法通过刷新校验，反之亦然。

<a id="12-claims-structure"></a>

### 1.2 Claims 结构

```go
// Access token
type AccessClaims struct {
    UserID   int    `json:"user_id"`
    Email    string `json:"email"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims // ExpiresAt / IssuedAt / NotBefore
}

// Refresh token
type RefreshClaims struct {
    UserID int    `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims // ExpiresAt / IssuedAt
}
```

签发时三个时间戳全部设置：`ExpiresAt`（过期）、`IssuedAt`（签发）、`NotBefore`（生效，等于 IssuedAt）。

<a id="13-security-hardening-golang-jwt-v5-source-verified"></a>

### 1.3 安全硬化（golang-jwt v5 源码确认）

**锁定签名算法（防 alg:none / 算法混淆）**：

golang-jwt v5 的 `ParseWithClaims` 默认接受任意算法。攻击者可构造 `alg:none` 的令牌绕过签名校验。必须在解析时显式锁定允许的算法：

```go
func validateToken(tokenString string, key []byte) (jwt.Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, newClaims(), func(*jwt.Token) (any, error) {
        return key, nil
    }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
    // ...
}
```

> 依据：`jwt/parser_option.go:11-19` —— `WithValidMethods` 的文档明确"heavily encouraged to prevent attacks such as algorithm confusion"。`jwt/parser.go:63-77` 校验令牌的 alg 必须在集合内，否则返回 `ErrTokenSignatureInvalid`。

**强制 exp claim**：

golang-jwt v5 默认在 `exp` 存在时校验，但**不要求**必须存在（无 exp 的令牌通过校验）。用 `WithExpirationRequired()` 强制：

```go
jwt.WithExpirationRequired()
```

> 依据：`jwt/parser_option.go:61-67` —— "By default exp claim is optional."。防御性编程要求显式强制，即使签发时总带 exp。

**HMAC 密钥要求**：

- 配置校验 `access_signing_key` / `refresh_signing_key` **≥32 字节**（256 bit）
- golang-jwt **不**校验最小密钥长度（`jwt/hmac.go` 源码确认），须由应用层强制
- 密钥必须是 `crypto/rand` 生成的随机字节，不能用人类可读字符串
- `config.example.toml` 提供生成命令：

```toml
# [REQUIRED] Generation command: openssl rand -base64 32
access_signing_key = "CHANGE_ME..."
refresh_signing_key = "CHANGE_ME..."
```

> 依据：`jwt/hmac.go:50-57` — "it is not advised to provide a []byte which was converted from a 'human readable' string... ideally be providing a []byte key which was produced from a cryptographically random source, e.g. crypto/rand."

<a id="14-access-token-blacklist"></a>

### 1.4 访问令牌黑名单

登出时将访问令牌的 SHA-256 哈希存入 `token_blacklist` 表，TTL 设为令牌的剩余有效期。中间件 `AuthWithBlacklist` 每次请求查询黑名单：

```go
// Logout
tokenHash := utils.HashToken(accessToken)
expiresAt := time.Now().Add(ttl) // ttl = the token's remaining lifetime
tokens.StoreBlacklistedToken(ctx, tokenHash, expiresAt)
```

短期访问令牌（24h）+ 黑名单的组合避免了维护服务端 session 的开销。令牌过期后黑名单记录自然失效，由定期清理回收。

---

<a id="2-refresh-token-rotation-and-reuse-detection"></a>

## 二、刷新令牌轮转与重用检测

<a id="21-single-use-rotation"></a>

### 2.1 一次性轮转

每次刷新都**吊销旧刷新令牌并签发新令牌对**（rotating refresh tokens）。刷新是一次性的，同一个刷新令牌不能用两次。

```
POST /auth/refresh { refresh_token }
  → validate the refresh token (DB lookup: not revoked + not expired)
  → revoke that refresh token (set revoked = true)
  → issue a new access + refresh token pair
```

<a id="22-soft-marker-revocation-the-revoked-column"></a>

### 2.2 软标记吊销（revoked 字段）

`refresh_tokens` 表带有 `revoked` bool 字段（通过 versioned migration 添加，默认 false）：

| 字段      | 类型 | 默认  | 说明                                |
| --------- | ---- | ----- | ----------------------------------- |
| `revoked` | bool | false | true 表示令牌已吊销（用于重用检测） |

吊销操作写 `UPDATE SET revoked = true`，不做物理 `DELETE`——保留的吊销记录使重用检测成为可能。

<a id="23-token-theft-reuse-detection"></a>

### 2.3 令牌盗用重用检测

当一个**已吊销**（`revoked=true`）的刷新令牌被再次提交时，判定为令牌被盗用，立即吊销该用户的**所有**刷新令牌：

```go
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) error {
    tokenHash := utils.HashToken(refreshToken)

    // 1. Look up a live token (revoked = false)
    record, err := s.tokens.GetRefreshToken(ctx, tokenHash) // WHERE revoked = false
    if err == domain.ErrNotFound {
        // 2. Is it a revoked token (revoked = true)? → reuse → theft
        revoked, err := s.tokens.IsRefreshTokenRevoked(ctx, tokenHash) // WHERE revoked = true
        if revoked {
            // token theft: revoke every refresh token of this user
            s.tokens.RevokeAllByUserID(ctx, record.UserID)
            return service.New(service.ErrInvalidToken, "refresh token reuse detected")
        }
        return service.New(service.ErrInvalidToken, "invalid refresh token")
    }
    // 3. Normal rotation: revoke the old pair, issue a new one
    // ...
}
```

**为什么需要软标记**：物理删除（`DELETE`）无法区分"令牌从来不存在"与"令牌存在过但已被吊销"。软标记保留了吊销记录，使两者可区分——前者是正常无效，后者是盗用信号。

<a id="24-query-rules"></a>

### 2.4 查询规则

- `GetRefreshToken`：`WHERE token_hash = ? AND revoked = false`（只返回有效令牌）
- `IsRefreshTokenRevoked`：`WHERE token_hash = ? AND revoked = true`（重用检测）
- `RevokeAllByUserID`：`UPDATE refresh_tokens SET revoked = true WHERE user_id = ? AND revoked = false`（盗用后全吊销）
- 过期 + revoked 的行仍可正常 prune 清理

---

<a id="3-oauth-github-same-page-redirect"></a>

## 三、OAuth（GitHub）：同页重定向

<a id="31-interaction-model-same-page-redirect"></a>

### 3.1 交互模式：同页重定向

markpost 采用**同页重定向**（模式 B），不采用弹窗模式。这是基于消除本质缺陷的决策：

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

<a id="32-complete-flow"></a>

### 3.2 完整流程

```
① User clicks "GitHub login"
   Frontend: GET /api/v1/oauth/url
   ──────────────────────────────────────────────
② Backend /oauth/url:
   - Generate state: crypto/rand 20 bytes → base64url
   - Generate verifier: oauth2.GenerateVerifier()
   - Store in ristretto: key=state, value={verifier, createdAt}, TTL=10min
   - Build the authorization URL: oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
   - Return { url, state }
   ──────────────────────────────────────────────
③ Frontend receives { url, state }:
   - Store expectedState in sessionStorage (key="oauth_state")
   - location.href = url (full-page navigation to GitHub)
   ──────────────────────────────────────────────
④ GitHub authorization page: the user signs in / authorizes
   ──────────────────────────────────────────────
⑤ GitHub redirects back to redirect_uri:
   Success: /auth/callback?code=xxx&state=yyy
   Failure: /auth/callback?error=access_denied
   ──────────────────────────────────────────────
⑥ Frontend /auth/callback page (static):
   - Parse code, state (or error) from the URL query
   - Failure branch (error present): show the error → router.replace('/login')
   - Success branch:
     a. Second check: the callback state === the expectedState in sessionStorage
     b. Clear oauth_state from sessionStorage
     c. POST /api/v1/oauth/login { code, state }
   ──────────────────────────────────────────────
⑦ Backend /oauth/login:
   - Validate state: look it up in ristretto → it must exist and be unexpired
   - Retrieve the verifier
   - Delete the state from ristretto immediately (one-time consumption, replay-proof)
   - oauth2.Exchange(ctx, code, oauth2.VerifierOption(verifier))
   - Fetch the GitHub user profile (/user + /user/emails)
   - GetOrCreateFromGitHub
   - completeLogin: issue the token pair
   - Return { user, token, refresh_token, expires_in }
   ──────────────────────────────────────────────
⑧ Frontend callback:
   - setAuth(token, user, refresh_token) → store in localStorage
   - router.replace('/dashboard')
```

<a id="33-state-validation-csrf-protection"></a>

### 3.3 State 校验（CSRF 防护）

**双层校验**：

| 层               | 存储           | 职责                                                         |
| ---------------- | -------------- | ------------------------------------------------------------ |
| 后端（主防线）   | ristretto 缓存 | 生成 state 时存，`/oauth/login` 时校验匹配 + 一次性消费      |
| 前端（二次校验） | sessionStorage | `/oauth/url` 返回的 state 存 sessionStorage，callback 时比对 |

后端是主防线（state 不匹配 → 401）。前端二次校验提前拦截，减少无效请求。

> **oauth2 库不做 state 校验**：`golang.org/x/oauth2` 的 `AuthCodeURL` 只是把 state 放进 URL，`Exchange` 不校验 state。文档明确："be sure to validate `http.Request.FormValue("state")`... this is the application's responsibility."（`oauth2.go:217-218`）。state 校验完全由应用层负责。

<a id="34-pkce-proof-key-for-code-exchange"></a>

### 3.4 PKCE（Proof Key for Code Exchange）

在 state 基础上加 PKCE 双保险：

| 组件        | 位置                                | 说明                                                                    |
| ----------- | ----------------------------------- | ----------------------------------------------------------------------- |
| `verifier`  | 随 state 同存 ristretto（不进 URL） | `oauth2.GenerateVerifier()`，32 字节随机                                |
| `challenge` | 授权 URL（`code_challenge` 参数）   | `S256ChallengeOption(verifier)` 计算 SHA256(verifier)                   |
| Exchange    | `VerifierOption(verifier)`          | 后端 Exchange 时传 verifier，GitHub 校验 SHA256(verifier) === challenge |

> 依据：`golang.org/x/oauth2/pkce.go` —— `GenerateVerifier()`（`pkce.go:27-38`）、`S256ChallengeOption`（`pkce.go:57-62`）、`VerifierOption`（`pkce.go:42-44`）。`oauth2.go:153-158` 建议用 PKCE 做 CSRF 防护。

**verifier 随 state 同存**：state 和 verifier 绑定，存为同一个 ristretto 条目（`key=state, value={verifier, createdAt}`）。`/oauth/login` 时用 state 查出 verifier，一次查询拿到两者。

<a id="35-state-and-verifier-storage"></a>

### 3.5 State 与 verifier 存储

存储介质：**ristretto 缓存**（项目已用，render cache 同款），TTL 10 分钟。

理由：state/verifier 是一次性、短生命周期的数据（生成后几分钟内被消费），内存缓存最合适，不增新依赖，不入 DB。

<a id="36-redirect_uri-and-the-callback-route"></a>

### 3.6 redirect_uri 与 callback 路由

GitHub OAuth App 的 `redirect_uri` 注册为：`https://<your-domain>/auth/callback`

前端路由 `/auth/callback`（放在 `(auth)` 路由组，用 PublicRoute 守卫）。

**不带 provider**：所有 provider（GitHub/Google/微信）共用同一个 `/auth/callback`。provider 信息编码在 state 里（后端存 `state → {provider, verifier}`），前端回调逻辑统一（取 code+state → POST → 存令牌 → 跳转）。扩展新 provider 是纯后端改动。

<a id="37-error-handling-matrix"></a>

### 3.7 错误处理矩阵

每个失败路径都有明确的用户可见行为：

| 失败场景                 | 检测点                                 | HTTP/状态                                     | 用户可见行为                                 |
| ------------------------ | -------------------------------------- | --------------------------------------------- | -------------------------------------------- |
| 用户拒绝授权             | GitHub 重定向带 `?error=access_denied` | callback 前端                                 | 提示"授权已取消" → `/login`                  |
| state 前端不匹配         | callback state ≠ sessionStorage        | callback 前端                                 | 提示"登录状态异常，请重试" → `/login`        |
| state 后端不匹配/过期    | ristretto 查不到 state                 | `/oauth/login` 401 `invalid_state`            | 提示"登录超时，请重试" → `/login`            |
| state 重复使用（重放）   | ristretto 已删除（一次性消费）         | `/oauth/login` 401 `invalid_state`            | 同上                                         |
| PKCE 校验失败            | Exchange 时 verifier 不匹配            | `/oauth/login` 401 `oauth_exchange_failed`    | 提示"授权验证失败" → `/login`                |
| GitHub Exchange 失败     | token endpoint 拒绝                    | `/oauth/login` 401 `oauth_exchange_failed`    | 同上                                         |
| 获取 GitHub 用户信息失败 | API 调用失败                           | `/oauth/login` 502 `github_user_fetch_failed` | 提示"无法获取 GitHub 账户信息" → `/login`    |
| code 缺失/格式错         | callback query 无 code                 | callback 前端                                 | 提示"授权回调无效" → `/login`                |
| 用户关 GitHub 页面/返回  | 无回调发生                             | 前端无感知                                    | 用户回到登录页需重新点击（无"卡在等待"状态） |
| 网络断开                 | POST /oauth/login fetch 失败           | callback 前端 catch                           | 显示"网络错误，请重试"，保留在 callback 页   |

<a id="38-oauth-error-codes"></a>

### 3.8 OAuth 错误码

定义在 `internal/service/auth/errors.go`（遵循 error-handling.md 的"域专属码分文件"原则）：

| ErrCode                  | Value                      | HTTP | 场景                                                                  |
| ------------------------ | -------------------------- | ---- | --------------------------------------------------------------------- |
| `ErrMissingState`        | `missing_state`            | 400  | `/oauth/login` 请求缺 state 参数                                      |
| `ErrMissingCode`         | `missing_code`             | 400  | `/oauth/login` 请求缺 code 参数                                       |
| `ErrInvalidState`        | `invalid_state`            | 401  | state 不匹配 / 过期 / 重放                                            |
| `ErrOAuthExchangeFailed` | `oauth_exchange_failed`    | 401  | PKCE 校验失败 或 GitHub Exchange 失败                                 |
| `ErrGitHubUserFetch`     | `github_user_fetch_failed` | 502  | 获取 GitHub 用户信息失败（上游故障，用 502 Bad Gateway 而非笼统 500） |

---

<a id="4-password-login"></a>

## 四、密码登录

<a id="41-hashing"></a>

### 4.1 哈希

使用 `golang.org/x/crypto/bcrypt`：

- **Cost**：`bcrypt.DefaultCost`（10）。有效范围 4–31，DefaultCost 是唯一推荐值。
- **盐**：`GenerateFromPassword` 内部用 `crypto/rand` 生成 16 字节随机盐，调用方无需提供。
- **校验**：`CompareHashAndPassword` 用 `crypto/subtle.ConstantTimeCompare` 做常量时间比较，防时序攻击。不匹配返回 `ErrMismatchedHashAndPassword`。

> 依据：`crypto/bcrypt/bcrypt.go:95-98`（GenerateFromPassword）、`bcrypt.go:153-154`（crypto/rand 盐）、`bcrypt.go:120`（常量时间比较）、`bcrypt.go:29`（ErrMismatchedHashAndPassword）。

<a id="42-password-length-policy"></a>

### 4.2 密码长度策略

| 约束     | 值         | 理由                                                                          |
| -------- | ---------- | ----------------------------------------------------------------------------- |
| 最小长度 | 8 字符     | NIST 800-63B 建议：长度比复杂度更重要                                         |
| 最大长度 | 72 字符    | bcrypt 算法限制（见下）                                                       |
| 复杂度   | **不强制** | NIST 800-63B 不推荐强制大小写+数字+符号（促使用户用可预测替换如 `P@ssw0rd!`） |

<a id="43-72-byte-ceiling-precheck"></a>

### 4.3 72 字节上限预检

bcrypt 算法只处理 ≤72 字节的密码。`GenerateFromPassword` 对超长密码返回 `ErrPasswordTooLong`（拒绝，不截断）。

> 依据：`bcrypt/bcrypt.go:96-98` —— `if len(password) > 72 { return nil, ErrPasswordTooLong }`。

在 `SetPassword` / `ChangePassword` 中**预检**长度，返回友好错误（而非 bcrypt 的原始 error）：

```go
if utf8.RuneCountInString(password) > 72 {
    return service.New(service.ErrValidation, "password must not exceed 72 characters")
}
```

注意用 `utf8.RuneCountInString`（按字符计数）而非 `len`（按字节），因为中文字符是多字节——72 个中文字符的字节数远超 72，但语义上是 72 个字符。bcrypt 的 72 字节限制是字节级的，所以实际校验要同时满足"字符数合理"且"字节数 ≤72"。

---

<a id="5-logout"></a>

## 五、登出

登出同时处理两种令牌：

| 令牌     | 登出操作                                                                                    |
| -------- | ------------------------------------------------------------------------------------------- |
| 访问令牌 | SHA-256 哈希存入 `token_blacklist`（TTL = 剩余有效期），中间件 `AuthWithBlacklist` 后续拒绝 |
| 刷新令牌 | `UPDATE refresh_tokens SET revoked=true WHERE user_id = ?`（吊销该用户的所有刷新令牌）      |

登出吊销刷新令牌，防止攻击者在访问令牌过期后用残留的刷新令牌重新获取访问权限。

---

<a id="6-frontend-token-storage-and-refresh"></a>

## 六、前端令牌存储与刷新

<a id="61-storage"></a>

### 6.1 存储

| 项       | 设计                                                                      |
| -------- | ------------------------------------------------------------------------- |
| 存储位置 | `localStorage`（key = `markpost_auth`）                                   |
| 存储内容 | `{ token, refreshToken, user, _hasHydrated }`                             |
| 状态管理 | Zustand + `persist` 中间件，`partialize` 只持久化 token/refreshToken/user |

<a id="62-xss-risk-and-mitigation"></a>

### 6.2 XSS 风险与缓解

localStorage 对所有同源 JS 可见，任何 XSS（含第三方库漏洞）都能窃取令牌。

**为什么接受 localStorage**：前端是纯静态客户端（无服务端运行时），没有可下发 Set-Cookie 的服务端，无法使用 HttpOnly cookie。访问令牌与刷新令牌都存 localStorage 是纯静态前端唯一可行的方案。

**缓解措施**：

- CSP（Content-Security-Policy）限制脚本来源
- 所有用户输入经过 bluemonday 消毒（文章渲染）+ 输出转义（模板）
- 依赖项定期审计

<a id="63-automatic-refresh-401-interception"></a>

### 6.3 自动刷新（401 拦截）

API client 拦截 401 响应，自动尝试刷新：

```typescript
// Pseudocode (see src/lib/api/base.ts)
async function handleTokenRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise; // single-flight: concurrent 401s share one refresh
  refreshPromise = refreshAccessToken().finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

// Inside the request function:
if (response.status === 401 && !skipAuthRefresh) {
  const refreshed = await handleTokenRefresh();
  if (!refreshed) throw new Error("Session expired");
  return retry(); // retry the original request with the new token
}
```

**单飞（single-flight）**：多个并发请求同时 401 时，只发一次刷新，所有请求共享结果（`refreshPromise` 去重）。避免刷新令牌被消耗多次（一次性轮转会拒绝第二次）。

刷新失败 → `logout()`（清空 localStorage）→ 后续请求无令牌 → 路由守卫重定向到 `/login`。

<a id="64-hydration-handling"></a>

### 6.4 水合处理

Zustand persist 从 localStorage 恢复是异步的。用 `_hasHydrated` 标志防止水合前用默认空状态（`token=null`）误判"未认证"导致闪烁跳转：

```typescript
onRehydrateStorage: () => (state) => {
  state?.setHasHydrated(true);
};
```

路由守卫在水合完成前渲染 PageSpinner，水合后根据真实认证状态决定渲染/重定向。

<a id="65-accept-language-header"></a>

### 6.5 Accept-Language 头

API client 在每个请求的 header 中携带 `Accept-Language: <当前 locale>`，后端据此返回对应语言的错误消息。详见 [frontend/i18n.md](./frontend/i18n.zh.md)。

---

<a id="7-oauth-callback-page-responsibilities"></a>

## 七、OAuth Callback 页面职责

`/auth/callback` 页面（`(auth)` 路由组，PublicRoute 守卫）的处理逻辑：

```typescript
// Pseudocode
function AuthCallbackPage() {
  const searchParams = useSearchParams();
  const setAuth = useAuthStore((s) => s.setAuth);

  useEffect(() => {
    const code = searchParams.get("code");
    const state = searchParams.get("state");
    const error = searchParams.get("error");

    // 1. Failure branch (GitHub returned an error)
    if (error) {
      router.replace("/login");
      return;
    }

    // 2. Parameter validation
    if (!code || !state) {
      router.replace("/login");
      return;
    }

    // 3. Frontend second state check
    const expectedState = sessionStorage.getItem("oauth_state");
    if (state !== expectedState) {
      router.replace("/login");
      return;
    }
    sessionStorage.removeItem("oauth_state");

    // 4. POST to the backend
    authApi
      .loginWithGitHub(code, state)
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

<a id="references"></a>

## 参考

- [error-handling.zh.md](./backend/error-handling.zh.md) —— ErrCode struct、错误响应格式、域专属码分文件
- [frontend/routes.md](./frontend/routes.zh.md) —— 路由守卫架构、安全边界声明
- [api-design.zh.md](./api-design.zh.md) —— `/oauth/*`、`/auth/*` 端点设计
