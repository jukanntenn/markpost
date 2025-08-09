# 需求文档

## 介绍

为了防止 create post 接口被恶意使用和滥用，需要实现多层次的限流机制。该功能将使用 ulule/limiter 第三方库来实现灵活的限流策略，支持基于 IP 地址和 post_key 的双重限制，并且通过配置文件来管理限制频率，便于运维调整。

## 需求

### 需求 1

**用户故事：** 作为系统管理员，我希望能限制单个 IP 地址的请求频率，以防止 DDoS 攻击和恶意行为。

#### 验收条件

1. WHEN 单个 IP 地址在 1 分钟内发起的 create post 请求超过配置的限制 THEN 系统 SHALL 返回 429 状态码并拒绝请求
2. WHEN 单个 IP 地址在 1 天内发起的 create post 请求超过配置的限制 THEN 系统 SHALL 返回 429 状态码并拒绝请求
3. WHEN IP 地址的请求频率在限制范围内 THEN 系统 SHALL 正常处理请求
4. WHEN 限制周期重置后 THEN 系统 SHALL 允许该 IP 地址重新发起请求

### 需求 2

**用户故事：** 作为系统管理员，我希望能基于 post_key 限制用户的发帖频率，以防止垃圾内容的大量发布。

#### 验收条件

1. WHEN 单个 post_key 在 1 分钟内发起的 create post 请求超过配置的限制 THEN 系统 SHALL 返回 429 状态码并拒绝请求
2. WHEN 单个 post_key 在 1 天内发起的 create post 请求超过配置的限制 THEN 系统 SHALL 返回 429 状态码并拒绝请求
3. WHEN post_key 的请求频率在限制范围内 THEN 系统 SHALL 正常处理请求
4. WHEN 限制周期重置后 THEN 系统 SHALL 允许该 post_key 重新发起请求

### 需求 3

**用户故事：** 作为系统管理员，我希望能通过配置文件灵活地调整限流参数，以便根据实际运营情况进行优化。

#### 验收条件

1. WHEN 系统启动时 THEN 系统 SHALL 从配置文件中读取所有限流相关的配置参数
2. IF 配置文件中不存在限流配置 THEN 系统 SHALL 使用合理的默认值
3. WHEN 配置文件中的限流参数被修改 THEN 系统 SHALL 在重启后应用新的配置
4. WHEN 配置参数格式错误或无效 THEN 系统 SHALL 记录错误并使用默认值

### 需求 4

**用户故事：** 作为开发者，我希望系统能集成 ulule/limiter 库，以便利用成熟的限流实现。

#### 验收条件

1. WHEN 系统处理限流逻辑时 THEN 系统 SHALL 使用 ulule/limiter 库的 API
2. WHEN 系统需要存储限流状态时 THEN 系统 SHALL 使用内存存储后端
3. WHEN ulule/limiter 库返回限流决策时 THEN 系统 SHALL 正确处理结果并返回相应的 HTTP 响应
4. WHEN 系统初始化限流器时 THEN 系统 SHALL 正确配置限流器的参数和内存存储

### 需求 5

**用户故事：** 作为 API 使用者，我希望收到清晰的错误信息，了解我被限流的原因和恢复时间。

#### 验收条件

1. WHEN 请求因 IP 限流被拒绝时 THEN 系统 SHALL 返回包含 "IP rate limit exceeded" 信息的 JSON 响应
2. WHEN 请求因 post_key 限流被拒绝时 THEN 系统 SHALL 返回包含 "Post key rate limit exceeded" 信息的 JSON 响应
3. WHEN 请求被限流时 THEN 系统 SHALL 在响应头中包含 X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset 等信息
4. WHEN 请求被限流时 THEN 系统 SHALL 返回 429 状态码
5. WHEN 请求成功处理时 THEN 系统 SHALL 在响应头中包含当前的限流状态信息

### 需求 6

**用户故事：** 作为系统管理员，我希望系统能记录限流事件，以便监控和分析系统使用情况。

#### 验收条件

1. WHEN 请求因限流被拒绝时 THEN 系统 SHALL 记录包含 IP 地址、post_key、限流类型和时间戳的日志
2. WHEN 系统检测到可疑的大量请求模式时 THEN 系统 SHALL 记录警告级别的日志
3. WHEN 限流配置发生变化时 THEN 系统 SHALL 记录配置变更信息
4. WHEN 限流系统出现错误时 THEN 系统 SHALL 记录错误信息但不影响正常请求的处理
