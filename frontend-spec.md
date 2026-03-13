# Frontend Specification

本文档定义了 AgentsMesh 项目的前端技术规范，包含两个独立的前端应用：

- **web** - 用户端应用（AI Agent Fleet Command Center）
- **web-admin** - 管理后台应用

---

## 1. 技术栈

### 1.1 核心框架

| 技术       | 版本   | 用途                         |
| ---------- | ------ | ---------------------------- |
| Next.js    | 16.1.6 | React 全栈框架（App Router） |
| React      | 19.2.3 | UI 库                        |
| TypeScript | ^5     | 类型安全                     |

### 1.2 UI 组件库

#### 共用组件库

| 技术                     | 版本     | 用途                   |
| ------------------------ | -------- | ---------------------- |
| Radix UI                 | ^1.x     | 无样式可访问性组件原语 |
| Lucide React             | ^0.577.0 | 图标库                 |
| Tailwind Merge           | ^3.5.0   | Tailwind 类名合并      |
| Class Variance Authority | ^0.7.1   | 组件变体管理           |
| clsx                     | ^2.1.1   | 条件类名               |
| next-themes              | ^0.4.6   | 主题切换（深色/浅色）  |
| sonner                   | ^2.0.7   | Toast 通知             |

#### web 特有组件

| 技术                   | 版本      | 用途               |
| ---------------------- | --------- | ------------------ |
| @blocknote/\*          | ^0.47.1   | 块编辑器（富文本） |
| @dnd-kit/\*            | ^6.x/10.x | 拖拽功能           |
| @use-gesture/react     | ^10.3.1   | 手势处理           |
| @xterm/\*              | ^6.0.0    | 终端模拟器         |
| @xyflow/react          | ^12.10.0  | 流程图/节点图      |
| recharts               | ^3.6.0    | 图表库             |
| cmdk                   | ^1.1.1    | 命令面板（Cmd+K）  |
| vaul                   | ^1.1.2    | 抽屉组件           |
| react-resizable-panels | ^4.3.3    | 可调整大小面板     |
| react-markdown         | ^10.1.0   | Markdown 渲染      |
| framer-motion          | ^12.35.0  | 动画库             |

#### web-admin 特有组件

| 技术                  | 版本    | 用途     |
| --------------------- | ------- | -------- |
| @tanstack/react-table | ^8.21.3 | 表格组件 |

### 1.3 样式方案

| 技术                    | 版本    | 用途               |
| ----------------------- | ------- | ------------------ |
| Tailwind CSS            | ^4.x    | 原子化 CSS 框架    |
| @tailwindcss/postcss    | ^4.x    | PostCSS 集成       |
| @tailwindcss/typography | ^0.5.19 | 排版插件（仅 web） |

**特点：** 采用 Tailwind CSS v4，使用 CSS-first 配置方式，无需 `tailwind.config.js` 文件。

### 1.4 状态管理

| 技术                  | 版本     | 用途                     |
| --------------------- | -------- | ------------------------ |
| Zustand               | ^5.0.x   | 轻量级状态管理（主要）   |
| @tanstack/react-query | ^5.90.16 | 服务端状态管理（仅 web） |

**状态管理模式：**

- Zustand 用于客户端状态（auth, pod, ticket 等）
- Persist 中间件持久化关键状态
- React Query 用于服务端缓存（web）

### 1.5 国际化

| 技术      | 版本   | 用途                     |
| --------- | ------ | ------------------------ |
| next-intl | ^4.7.0 | Next.js 国际化（仅 web） |

**支持语言：** en, zh, ja, ko, de, fr, es, pt

### 1.6 数据可视化

| 技术          | 版本     | 用途                      |
| ------------- | -------- | ------------------------- |
| @xyflow/react | ^12.10.0 | Mesh 网络可视化（仅 web） |
| recharts      | ^3.6.0   | 统计图表（仅 web）        |

### 1.7 实时通信

| 技术                     | 用途                         |
| ------------------------ | ---------------------------- |
| WebSocket                | 实时事件推送（仅 web）       |
| EventSubscriptionManager | 连接管理、心跳检测、断线重连 |

### 1.8 分析监控

| 技术    | 版本     | 用途               |
| ------- | -------- | ------------------ |
| PostHog | ^1.347.2 | 产品分析（仅 web） |

### 1.9 工具库

| 技术        | 版本   | 用途                                |
| ----------- | ------ | ----------------------------------- |
| date-fns    | ^4.1.0 | 日期处理（仅 web）                  |
| gray-matter | ^4.0.3 | Markdown Frontmatter 解析（仅 web） |
| remark-gfm  | ^4.0.1 | GitHub 风格 Markdown（仅 web）      |

### 1.10 构建工具

| 技术               | 用途                |
| ------------------ | ------------------- |
| Next.js 内置打包   | 生产构建            |
| Turbopack          | 开发模式加速（web） |
| Output: standalone | Docker 部署优化     |

### 1.11 代码质量

| 技术               | 版本   | 用途                |
| ------------------ | ------ | ------------------- |
| ESLint             | ^9     | 代码检查            |
| eslint-config-next | 16.1.x | Next.js ESLint 规则 |
| TypeScript         | ^5     | 静态类型检查        |

### 1.12 测试

| 技术                        | 版本       | 用途              |
| --------------------------- | ---------- | ----------------- |
| Vitest                      | ^4.0.x     | 测试运行器        |
| @vitest/coverage-v8         | ^4.0.x     | 代码覆盖率        |
| @testing-library/react      | ^16.3.x    | React 测试工具    |
| @testing-library/jest-dom   | ^6.9.1     | DOM 断言扩展      |
| @testing-library/user-event | ^14.6.1    | 用户交互模拟      |
| @vitejs/plugin-react        | ^5.x       | Vitest React 插件 |
| jsdom                       | ^27.x/28.x | DOM 模拟环境      |

---

## 2. 目录结构

### 2.1 web 项目结构

```
web/
├── src/
│   ├── app/                      # Next.js App Router 页面
│   │   ├── (auth)/              # 认证路由组（未登录）
│   │   │   ├── auth/            # OAuth 回调
│   │   │   ├── forgot-password/ # 忘记密码
│   │   │   ├── invite/          # 邀请注册
│   │   │   ├── login/           # 登录
│   │   │   ├── onboarding/      # 新用户引导
│   │   │   ├── register/        # 注册
│   │   │   ├── reset-password/  # 重置密码
│   │   │   ├── runners/         # Runner 注册
│   │   │   └── verify-email/    # 邮箱验证
│   │   ├── (dashboard)/         # 仪表盘路由组（已登录）
│   │   │   ├── [org]/           # 动态组织路由
│   │   │   │   ├── channels/    # 频道/消息
│   │   │   │   ├── loops/       # Loop 自动化
│   │   │   │   ├── mesh/        # Mesh 可视化
│   │   │   │   ├── repositories/# 仓库管理
│   │   │   │   ├── runners/     # Runner 管理
│   │   │   │   ├── settings/    # 组织设置
│   │   │   │   ├── tickets/     # Ticket 面板
│   │   │   │   └── workspace/   # 工作区/IDE
│   │   │   ├── settings/        # 用户设置
│   │   │   ├── support/         # 支持
│   │   │   └── layout.tsx       # Dashboard 布局
│   │   ├── about/               # 关于页面
│   │   ├── blog/                # 博客
│   │   ├── careers/             # 招聘
│   │   ├── changelog/           # 更新日志
│   │   ├── demo/                # 演示
│   │   ├── docs/                # 文档
│   │   ├── enterprise/          # 企业版
│   │   ├── privacy/             # 隐私政策
│   │   ├── terms/               # 服务条款
│   │   ├── globals.css          # 全局样式 + Tailwind 配置
│   │   ├── layout.tsx           # 根布局
│   │   ├── page.tsx             # 首页
│   │   ├── opengraph-image.tsx  # OG 图片生成
│   │   ├── robots.ts            # robots.txt
│   │   └── sitemap.ts           # 站点地图
│   ├── components/              # React 组件
│   │   ├── autopilot/           # Autopilot 控制器
│   │   ├── billing/             # 计费/订阅
│   │   ├── channel/             # 频道/消息
│   │   ├── common/              # 通用组件
│   │   ├── docs/                # 文档组件
│   │   ├── i18n/                # 国际化组件
│   │   ├── icons/               # 自定义图标
│   │   ├── ide/                 # IDE 界面组件
│   │   ├── landing/             # 落地页组件
│   │   ├── layout/              # 布局组件
│   │   ├── loops/               # Loop 自动化组件
│   │   ├── mesh/                # Mesh 可视化组件
│   │   ├── mobile/              # 移动端组件
│   │   ├── pod/                 # Pod 管理组件
│   │   ├── promo-code/          # 优惠码组件
│   │   ├── pwa/                 # PWA 组件
│   │   ├── repository/          # 仓库管理组件
│   │   ├── settings/            # 设置组件
│   │   ├── shared/              # 共享组件
│   │   ├── support/             # 支持组件
│   │   ├── theme/               # 主题组件
│   │   ├── tickets/             # Ticket 面板组件
│   │   ├── ui/                  # 基础 UI 组件（shadcn 风格）
│   │   └── workspace/           # 工作区组件
│   ├── content/                 # 静态内容
│   │   └── blog/                # 博客 Markdown 文件
│   ├── hooks/                   # 自定义 Hooks
│   │   ├── useAsyncData.ts      # 异步数据加载
│   │   ├── useBrowserNotification.ts # 浏览器通知
│   │   ├── useChannelChat.ts    # 频道聊天
│   │   ├── useLemonSqueezy.ts   # 支付集成
│   │   ├── useMentionCandidates.ts # @提及候选
│   │   ├── usePodStatus.ts      # Pod 状态
│   │   ├── useRealtimeEvents.ts # 实时事件
│   │   ├── useServerUrl.ts      # 服务器 URL
│   │   ├── useTerminal.ts       # 终端
│   │   ├── useTicketPrefetch.ts # Ticket 预加载
│   │   └── useTouchScroll.ts    # 触摸滚动
│   ├── i18n/                    # i18n 配置
│   │   └── request.ts           # next-intl 请求配置
│   ├── lib/                     # 核心工具库
│   │   ├── api/                 # API 客户端
│   │   │   ├── base.ts          # 基础请求封装
│   │   │   └── *.ts             # 各领域 API
│   │   ├── i18n/                # i18n 工具
│   │   ├── realtime/            # WebSocket 实时通信
│   │   │   └── EventSubscriptionManager.ts
│   │   ├── utils/               # 工具函数
│   │   ├── env.ts               # 环境变量
│   │   ├── theme.ts             # 主题
│   │   └── utils.ts             # 通用工具
│   ├── messages/                # 翻译文件
│   │   ├── de/                  # 德语
│   │   ├── en/                  # 英语
│   │   ├── es/                  # 西班牙语
│   │   ├── fr/                  # 法语
│   │   ├── ja/                  # 日语
│   │   ├── ko/                  # 韩语
│   │   ├── pt/                  # 葡萄牙语
│   │   └── zh/                  # 中文
│   ├── providers/               # React Context Providers
│   │   ├── PostHogProvider.tsx  # 分析
│   │   └── RealtimeProvider.tsx # 实时事件
│   ├── stores/                  # Zustand 状态仓库
│   │   ├── auth.ts              # 认证状态
│   │   ├── autopilot.ts         # Autopilot 状态
│   │   ├── channel.ts           # 频道状态
│   │   ├── gitProvider.ts       # Git Provider
│   │   ├── ide.ts               # IDE 状态
│   │   ├── loop.ts              # Loop 状态
│   │   ├── mesh.ts              # Mesh 状态
│   │   ├── organization.ts      # 组织状态
│   │   ├── pod.ts               # Pod 状态
│   │   ├── repository.ts        # 仓库状态
│   │   ├── runner.ts            # Runner 状态
│   │   ├── terminalConnection.ts# 终端连接
│   │   ├── ticket.ts            # Ticket 状态
│   │   ├── user.ts              # 用户状态
│   │   └── workspace.ts         # 工作区状态
│   ├── test/                    # 测试配置
│   │   └── setup.ts             # Vitest 设置
│   └── types/                   # TypeScript 类型定义
│       └── next-intl.d.ts
├── public/                      # 静态资源
├── next.config.ts               # Next.js 配置
├── tsconfig.json                # TypeScript 配置
├── postcss.config.mjs           # PostCSS 配置
├── eslint.config.mjs            # ESLint 配置
├── vitest.config.ts             # Vitest 配置
├── package.json                 # 依赖配置
└── pnpm-lock.yaml               # 锁文件
```

### 2.2 web-admin 项目结构

```
web-admin/
├── src/
│   ├── app/                      # Next.js App Router 页面
│   │   ├── layout.tsx           # 根布局
│   │   ├── globals.css          # 全局样式 + Tailwind
│   │   ├── page.tsx             # 首页重定向
│   │   ├── login/               # 登录页
│   │   │   ├── page.tsx
│   │   │   └── __tests__/
│   │   └── (dashboard)/         # 仪表盘路由组（需认证）
│   │       ├── layout.tsx       # Dashboard 布局（认证守卫）
│   │       ├── page.tsx         # 仪表盘首页（统计概览）
│   │       ├── __tests__/
│   │       ├── audit-logs/      # 审计日志
│   │       ├── organizations/    # 组织管理
│   │       ├── promo-codes/     # 优惠码管理
│   │       ├── relays/          # Relay 服务器管理
│   │       ├── runners/         # Runner 实例管理
│   │       ├── skill-registries/# Skill 注册表管理
│   │       ├── support-tickets/ # 支持工单
│   │       └── users/           # 用户管理
│   ├── components/              # React 组件
│   │   ├── theme-provider.tsx   # 主题 Provider
│   │   ├── layout/              # 布局组件
│   │   │   ├── header.tsx       # 页头
│   │   │   └── sidebar.tsx      # 侧边导航栏
│   │   └── ui/                  # 基础 UI 组件（shadcn 风格）
│   │       ├── badge.tsx
│   │       ├── button.tsx
│   │       ├── card.tsx
│   │       ├── dialog.tsx
│   │       ├── dropdown-menu.tsx
│   │       ├── input.tsx
│   │       ├── label.tsx
│   │       ├── select.tsx
│   │       ├── sheet.tsx
│   │       ├── table.tsx
│   │       └── textarea.tsx
│   ├── lib/                     # 工具库
│   │   ├── utils.ts             # 工具函数
│   │   ├── support-constants.ts # 支持工单常量
│   │   └── api/                 # API 客户端
│   │       ├── base.ts          # 基础请求封装
│   │       └── admin.ts         # 管理后台 API
│   ├── stores/                  # Zustand 状态仓库
│   │   └── auth.ts              # 认证状态
│   └── test/                    # 测试配置
│       └── setup.ts             # Vitest 设置
├── next.config.ts               # Next.js 配置
├── tsconfig.json                # TypeScript 配置
├── postcss.config.mjs           # PostCSS 配置
├── eslint.config.mjs            # ESLint 配置
├── vitest.config.ts             # Vitest 配置
├── package.json                 # 依赖配置
└── pnpm-lock.yaml               # 锁文件
```

---

## 3. 框架配置

### 3.1 Next.js 配置

#### web (next.config.ts)

```typescript
const nextConfig: NextConfig = {
  output: "standalone",  // Docker 部署优化

  // 开发模式 Turbopack 加速
  turbopack: { ... },

  // 国际化插件
  experimental: {
    turbo: { ... }
  }
};

// next-intl 插件集成
const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

// 开发环境 API 代理
rewrites: async () => [
  { source: "/api/:path*", destination: "http://localhost:8080/api/:path*" }
]

// 环境变量映射
env: {
  NEXT_PUBLIC_PRIMARY_DOMAIN: process.env.PRIMARY_DOMAIN,
  NEXT_PUBLIC_USE_HTTPS: process.env.USE_HTTPS,
}
```

#### web-admin (next.config.ts)

```typescript
const nextConfig: NextConfig = {
  output: "standalone",  // Docker 部署优化

  // 图片域名配置
  images: {
    remotePatterns: [
      { protocol: "http", hostname: "**" },
      { protocol: "https", hostname: "**" }
    ]
  }
};

// 开发环境 API 代理
rewrites: async () => [
  { source: "/api/:path*", destination: "http://localhost:8080/api/:path*" }
]

// 环境变量映射
env: {
  NEXT_PUBLIC_PRIMARY_DOMAIN: process.env.PRIMARY_DOMAIN,
  NEXT_PUBLIC_USE_HTTPS: process.env.USE_HTTPS,
}
```

### 3.2 TypeScript 配置

两个项目共用相同配置模式：

```jsonc
{
  "compilerOptions": {
    "target": "ES2017",
    "lib": ["dom", "dom.iterable", "esnext"],
    "module": "esnext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "noEmit": true,
    "incremental": true,
    "isolatedModules": true,
    "paths": {
      "@/*": ["./src/*"]
    },
    "plugins": [{ "name": "next" }]
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

### 3.3 Tailwind CSS 配置

采用 Tailwind CSS v4 的 CSS-first 配置方式：

```css
/* globals.css */
@import "tailwindcss";

@theme {
  --color-background: oklch(100% 0 0);
  --color-foreground: oklch(14.9% 0.024 285.8);
  --color-primary: oklch(62.2% 0.175 292.1);
  /* ... 更多主题变量 */
}

:root {
  --background: var(--color-background);
  --foreground: var(--color-foreground);
  /* ... CSS 变量映射 */
}

.dark {
  --background: oklch(14.9% 0.024 285.8);
  --foreground: oklch(97.1% 0.007 247.9);
  /* ... 深色主题 */
}
```

### 3.4 PostCSS 配置

```javascript
// postcss.config.mjs
export default {
  plugins: {
    "@tailwindcss/postcss": {},
  },
};
```

### 3.5 ESLint 配置

```javascript
// eslint.config.mjs
import { dirname } from "path";
import { fileURLToPath } from "url";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

const eslintConfig = [
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  {
    ignores: [".next/**", "out/**", "build/**", "next-env.d.ts"],
  },
];

export default eslintConfig;
```

### 3.6 Vitest 配置

```typescript
// vitest.config.ts
export default defineConfig({
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html", "cobertura", "junit"],
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  plugins: [react()],
});
```

### 3.7 环境变量

#### web

| 变量                         | 用途           |
| ---------------------------- | -------------- |
| `PRIMARY_DOMAIN`             | 主域名         |
| `USE_HTTPS`                  | 是否使用 HTTPS |
| `NEXT_PUBLIC_PRIMARY_DOMAIN` | 客户端域名     |
| `NEXT_PUBLIC_USE_HTTPS`      | 客户端协议     |

#### web-admin

| 变量                         | 用途           |
| ---------------------------- | -------------- |
| `PRIMARY_DOMAIN`             | 主域名         |
| `USE_HTTPS`                  | 是否使用 HTTPS |
| `NEXT_PUBLIC_GITLAB_SSO_URL` | GitLab SSO URL |

---

## 4. 架构模式

### 4.1 路由架构

```
App Router (Next.js 16)
    │
    ├── (auth) Route Group ──────► 未认证页面
    │       │
    │       └── login, register, forgot-password, etc.
    │
    └── (dashboard) Route Group ──► 已认证页面
            │
            ├── RealtimeProvider (web) ──► WebSocket 连接
            │
            └── [org] Dynamic Route ──► 组织上下文
                    │
                    ├── workspace/ ──► IDE Shell
                    ├── pods/ ──────► Pod 管理
                    ├── mesh/ ──────► Mesh 可视化
                    ├── tickets/ ───► Ticket 面板
                    ├── channels/ ──► 消息
                    └── loops/ ─────► Loop 自动化
```

### 4.2 状态管理架构

```
Zustand Store Pattern
    │
    ├── persist() 中间件 ──► localStorage 持久化
    │
    ├── partialize() ──► 选择性持久化
    │
    └── Store 结构
            │
            ├── State (数据)
            └── Actions (方法)
                    │
                    ├── Setters (更新)
                    ├── Fetchers (获取)
                    └── Mutations (变更)
```

### 4.3 API 客户端架构

```
lib/api/
    │
    ├── base.ts ──► 核心请求封装
    │       │
    │       ├── request<T>() ──► 泛型请求方法
    │       ├── Auth Header 注入
    │       ├── Token 刷新处理
    │       └── 错误处理
    │
    └── [domain].ts ──► 领域 API
            │
            ├── auth.ts
            ├── pod.ts
            ├── ticket.ts
            └── ...
```

### 4.4 实时事件架构 (web)

```
WebSocket 连接
    │
    └── EventSubscriptionManager
            │
            ├── 连接管理
            ├── 心跳检测 (ping/pong)
            ├── 断线重连 (指数退避)
            └── 事件订阅分发
                    │
                    └── RealtimeProvider
                            │
                            ├── Pod 事件 ──► podStore
                            ├── Runner 事件 ──► runnerStore
                            ├── Ticket 事件 ──► ticketStore
                            └── Channel 事件 ──► channelStore
```

---

## 5. 组件设计规范

### 5.1 UI 组件 (shadcn/ui 风格)

- 基于 Radix UI 原语构建
- 使用 `class-variance-authority` 管理变体
- 使用 `tailwind-merge` + `clsx` 合并类名
- 通过 `Slot` 支持多态组件

```typescript
// Button 组件示例
const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md text-sm font-medium",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive: "bg-destructive text-destructive-foreground",
        outline: "border border-input bg-background hover:bg-accent",
        secondary: "bg-secondary text-secondary-foreground",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-10 px-4 py-2",
        sm: "h-9 rounded-md px-3",
        lg: "h-11 rounded-md px-8",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);
```

### 5.2 组件目录组织

| 目录         | 用途                                     |
| ------------ | ---------------------------------------- |
| `ui/`        | 基础 UI 原语（Button, Input, Dialog 等） |
| `shared/`    | 跨功能共享组件                           |
| `[feature]/` | 功能特定组件（pod, mesh, tickets 等）    |
| `layout/`    | 布局组件（ResponsiveShell, Sidebar 等）  |

---

## 6. 主题系统

### 6.1 CSS 变量体系

```css
:root {
  --background: /* 浅色背景 */ ;
  --foreground: /* 浅色前景 */ ;
  --primary: /* 主色 */ ;
  --secondary: /* 次色 */ ;
  --accent: /* 强调色 */ ;
  --muted: /* 静音色 */ ;
  --destructive: /* 危险色 */ ;
  --border: /* 边框色 */ ;
  --ring: /* 焦点环色 */ ;
}

.dark {
  --background: /* 深色背景 */ ;
  --foreground: /* 深色前景 */ ;
  /* ... 深色主题变量 */
}
```

### 6.2 主题切换

使用 `next-themes` 实现：

```typescript
// ThemeProvider
import { ThemeProvider } from "next-themes";

<ThemeProvider attribute="class" defaultTheme="dark">
  {children}
</ThemeProvider>;
```

---

## 7. 测试规范

### 7.1 测试工具链

- **Vitest** - 测试运行器
- **@testing-library/react** - React 测试工具
- **@testing-library/user-event** - 用户交互模拟
- **jsdom** - DOM 环境

### 7.2 测试文件组织

```
src/
├── components/
│   └── ui/
│       └── button.tsx
│       └── __tests__/
│           └── button.test.tsx
├── lib/
│   └── utils.ts
│   └── __tests__/
│       └── utils.test.ts
└── test/
    └── setup.ts  # 全局测试设置
```

### 7.3 测试设置

```typescript
// test/setup.ts
import "@testing-library/jest-dom";

// 全局 Mock
global.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}));
```

---

## 8. 部署配置

### 8.1 构建输出

- `output: "standalone"` - Docker 部署优化
- 独立输出，无需 node_modules

### 8.2 Docker 配置

参考 `ci/web.Dockerfile` 和 `ci/web-admin.Dockerfile`

---

## 9. 包管理

使用 **pnpm** 作为包管理器：

- `pnpm-workspace.yaml` - Monorepo 工作区配置
- `pnpm-lock.yaml` - 锁文件
- `.npmrc` - npm 配置
