# MRFC: prek is the single source of format and lint

Status: implemented

[English](2026-08-12-prek-single-source-of-format-and-lint.md) | 中文

## Problem

格式化器与 linter 的调用曾并行定义：AI 后置编辑 hook 跑一组命令，CI 跑另一组，开发者的肌肉记忆是第三组。定义漂移了 —— 一条裸 `prettier` 条目依赖只有部分机器才有的环境二进制，"跑一下 linter" 在不同语境下含义不同，于是格式化打架与 lint 跳过在本地复现。

## Decision

管线运行的每个检查都恰好声明一次，在 prek 里：根部的 `prek.toml`（内建只读检查、格式化器、部署文件闸门、Conventional Commits）、`backend/prek.toml`（Go fmt/lint/generate 闸门）和 `frontend/prek.toml`（ESLint/Prettier）。调用契约固定：AI 后置编辑 hook 跑 `prek run --group fmt --files <edited>`，AI Stop hook 跑 `prek run --group lint --all-files`，CI 的 Lint 作业跑 `prek run --all-files` —— 只安装这些 hook 调用的工具链。`git commit` 跑 pre-commit 阶段，`commit-msg` 检查消息，pre-push 跑测试。这些文件之外不存在任何平行的格式化器或 lint 定义；依赖可选工具的 hook 带一条通知优雅跳过。

## Alternatives considered

**每个环境一套 Make/just 目标。** 有单一可看之处，但没有任何东西把 AI hook、CI 和人钉在同一批目标上 —— 正是本决策要移除的漂移。

**lefthook / husky。** 有能力的 hook 运行器；选 prek 是为原生多语言 hook 管理与配置校验，在它旁边再放一个运行器会重新引入正被消灭的平行定义。

**只在 CI 强制。** 定义单一，但反馈迟到几分钟，本地提交积累未格式化的工作；分层模型（提交时快速修复器、CI 里全套件）无论如何都需要本地 hook。

## Consequences

`prek install` 是一次性安装，之后每个运行这些 hook 的环境看到相同行为。加一个检查意味着编辑一个 prek 文件（并在 CI 提供其工具链） —— 本批次加的文档闸门走的正是这条路。可选工具 hook 必须守住优雅跳过契约，使缺失的本地二进制退化为一条通知而非被阻塞的提交。提交要求 hook 链在场；`--no-verify` 出界。
