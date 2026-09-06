# MRFC: GitHub 身份标识进入 admin 用户页

Status: implemented

[English](2026-09-05-admin-github-identity.md) | 中文

## Problem

用户模型自 OAuth 落地起就带有 GitHub 身份——注册时写入 `github_id` 与 `avatar_url`——但这些信息从未以“身份”的形式到达 admin 管理台的用户界面。用户列表只有裸用户名（加 VIP）；详情页只有一行纯文本 "GitHub: Linked as #N"。管理员无法一眼判断账号的登录方式，也无法视觉上认出一个用户；`avatar_url` 更是存而不用——admin API 根本没把它带出去。

## Decision

admin 用户页在用户名旁渲染头像 + 登录方式标记：

- `AdminUserItem`（后端 DTO，`newAdminUserItem`）新增 `avatar_url`——从既有模型列组装的增量 wire-format 字段；无 schema 变更、无迁移。
- 两个 `ui/` 小组件负责展示：`UserAvatar`（GitHub 头像；密码注册用户回退为首字母圆）与 `GithubBadge`（仅图标的 lucide `GithubIcon` 标记，带原生 "GitHub" tooltip，与 VIP 徽章一样 locale 不变）。
- 出现位置：admin 用户列表（桌面表格与移动卡片）每行用户名旁、详情页页眉、详情页 "GitHub" 资料行的 "Linked as #N" 旁。

不提供 GitHub 主页外链：主页 URL 需要 schema 里没有存的 GitHub login，而 `github_id` 没有公开的主页 URL。

## Alternatives considered

**仅图标徽章，不放头像。** “GitHub 标识”的最小读法。它标记了登录方式，却丢掉了 GitHub 用户真正认得的识别信号——头像，而数据库里就有。展示 avatar_url 是这个字段存在的本意，是本质修复而非范围蔓延。

**存储 GitHub login 并外链。** 能给出真正的主页链接，但需要 schema 迁移（ask-first 级变更），换来的只是管理员从控制台跳去第三方站点——成本高、治理价值低。若将来管理员需要核查上游 GitHub 账号本身，再重新评估。

**全应用展示头像（设置页、菜单）。** 需求只针对 admin 管理台的用户相关页面；扩大范围会触及没人要求过的界面。

## Consequences

管理员在列表与详情页都能看到登录方式与视觉身份；密码用户靠首字母圆保持布局稳定。头像从 `avatars.githubusercontent.com` 加载——admin 浏览器为每个渲染的 GitHub 用户发起一次外部图片请求，对管理页可接受，且回退圆不依赖加载结果。DTO 字段是增量的：旧客户端忽略之，Swagger 文档随 `go generate` 再生成。
