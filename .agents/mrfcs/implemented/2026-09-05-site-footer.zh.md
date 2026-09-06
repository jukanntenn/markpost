# MRFC: 着陆页与应用壳的站点 footer

Status: implemented

[English](2026-09-05-site-footer.md) | 中文

## Problem

只有着陆页有 footer——colophon：一个收尾 CTA、一行居中的三链接、版权/排印行。应用壳（dashboard 及全部 admin 页）在最后一张卡片处戛然而止。运行版本在 UI 里无处可寻：自托管用户无法从控制台判断实例跑的是哪个 release，报障时也带不上版本号。着陆页 footer 也只为项目真实目的地中的三个提供入口。

## Decision

两块 footer，对标 GitHub/Vercel 的 footer 骨架（品牌区 + 链接列 + 底条），按实际存在的链接缩减规模：

- **着陆页（`SiteFooter.tsx`，取代 `ColophonFooter`）**：收尾 CTA 保留在顶部；其下是品牌区（logo + 一句话 tagline）与两列链接——*资源*（文档、问题反馈）与*项目*（GitHub、Docker Hub、MIT License）——底条为 `© {year} markpost · MIT License · v{APP_VERSION}`，排印说明保留在右侧。移动端链接列堆叠。
- **应用壳（`AppShell.tsx`）**：钉在页面底部的 slim footer——左侧 `markpost v{APP_VERSION}`，右侧文档 + GitHub 链接，与顶栏同一 1200px 容器对齐。auth 页不加 footer（登录卡片本就有回首页链接）。
- **版本来源**：`src/lib/site.ts` 的 `APP_VERSION` 从 `package.json` 导入 `version`——release 流程 bump 的就是它；唯一来源，别处不得硬编码。站点级链接常量（`REPO_URL`、`DOCS_URL`、`ISSUES_URL`、`LICENSE_URL`、`DOCKER_HUB_URL`）从 `components/landing/links.ts` 迁到这里，后者只保留 `LANDING_CONTAINER`。
- 文案：`landing.colophon` 新增 `tagline`/`resources`/`project`/`issues`；新增 `footer` 命名空间（dashboard 链接标签）——均覆盖全部四个 locale。

不发明目的地：状态页、支持、社交账号并不存在，footer 里就没有它们。

## Alternatives considered

**维持 colophon 作 footer。** colophon 是刻意的书籍设计签名，但维护者的明确方向是“对标知名服务的专业 footer”；存续本身不是权威。排印说明留在底条——签名没有丢，只是退位。

**单行链接 footer（β 方案）。** 更轻，但“品牌区 + 链接列”才是读起来像专业 footer 的结构；只有五个真实链接时，专业感来自布局而非凑列。

**版本号取自后端 `/api/v1/version`。** 该端点报告二进制的 git-describe 串；前端是构建期定死的静态导出，打包进 bundle 的 `package.json` 版本才是构建自身的真相——离线可用，也不必在每次壳渲染时发起请求竞态。

**auth 页也加 footer。** 聚焦的登录卡片下再放 footer 只是噪音；卡片已带 markpost 回家链接。

## Consequences

着陆页承载完整 footer；控制台显示自己的运行版本——自托管运维真正需要的那块。`specs/frontend/routes.md`（+ 中文对）§06 随同一变更重写。两个代价：footer 链接集里的目的地若失效需剪除（现在都集中在 `src/lib/site.ts` 一个文件）；着陆页的 `landing.colophon` 命名空间同时装着 CTA 与 footer 文案——因二者渲染于同一组件而接受，CTA key（`heading`/`subheading`/`getStarted`）与 footer key（列、底条）在其内部仍界限分明。
