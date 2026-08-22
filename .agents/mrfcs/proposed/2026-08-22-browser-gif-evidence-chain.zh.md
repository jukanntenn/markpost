# MRFC: Browser GIF evidence for UI changes

Status: proposed

English | [中文](2026-08-22-browser-gif-evidence-chain.md)

## Problem

[开发闭环](2026-08-22-agent-driven-development-loop.zh.md)以 Playwright 截图作为 UI 变更层的证据标准：截图证明一个状态渲染正确，仅此而已。交互恰是截图装不下的东西——过渡、动画时序、hover 与 focus 行为、多步流程——而 UI 回归也正藏在那里。参考实现对涉及 GUI 的 pull request 录制一段简短的浏览器 GIF，把录像当作证据链的一环；markpost 的 v1 有意推迟了这套机制，本记录持有升级路径与启用触发条件。

## Proposal

每个改变用户可见前端行为的实施层，在既有截图之外附一段简短的屏幕录像（几秒钟，GIF 或无声 MP4），对着 dev 环境演练被改动的交互：动作执行、过渡呈现、状态落定。录制复用本就必需的 Playwright 验证会话（`playwright-cli` 视频捕获，或由截图序列经 `ffmpeg` 合成），由 `dev-loop` 的验证步骤脚本化，成本是一个开关而非一项手工杂务。证据落在 pull request body 里截图旁侧；录像绝不提交进仓库。闭环的验收映射为每个 UI 层加一行：演示的交互、录像的链接。非 UI 层不受影响。

## Alternatives considered

**只用截图。** 零新增成本，但交互性主张在人类亲手点验之前始终是未经验证的文字——而这正是闭环从人类一天里移除的那种 review。

**带解说的完整视频。** 证据更丰富，每个 pull request 的制作成本不成比例，且无法一眼审完；一段简短无声剪辑已承载增量。

**交互式评审环境（每个 pull request 的临时预览部署）。** 最强的证据与最重的机制——为单镜像静态导出的前端建预览基础设施；远超交互验证所需。

## Acceptance criteria

改变用户可见交互的实施层，没有嵌入演示该交互的录像就过不了 agent 预审；录制步骤是验证的脚本化部分而非临场判断；没有用户可见交互变更的层在证据区显式标注为如此。

## Risks

若无上限，录像会撑大 pull request body——几秒钟规则与 skill 里的体积上限保持其小巧，body 内嵌体验劣化时回退到 issue 评论附件托管。动画时序的捕获抖动增加验证重试成本。而建造它的触发条件本身，是 review 期间人类反复亲手重验交互主张；在该信号出现之前，截图仍是受认标准，本提案等待。
