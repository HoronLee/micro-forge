# AGENTS.md - servora Web 前端

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-03-06 -->

## 目录定位

前端已迁入 `app/servora/service/web/`。这是一个 Vue 3 + TypeScript + Vite 项目，和 `servora` 服务同仓协作。

## 当前技术栈

- Vue 3
- Vite 7
- TypeScript
- Pinia
- Vue Router
- Element Plus
- Tailwind CSS 4（通过 `@tailwindcss/vite`）
- Vitest
- Playwright
- `unplugin-auto-import` / `unplugin-vue-components`

## 当前结构

```text
web/
├── e2e/
├── public/
├── src/
│   ├── __tests__/
│   ├── router/
│   ├── service/
│   │   └── gen/
│   ├── stores/
│   ├── utils/
│   └── views/
├── package.json
├── vite.config.ts
├── playwright.config.ts
└── vitest.config.ts
```

## 生成与联调

- 协议客户端生成脚本：`bun run gen:proto`
- 当前脚本实际调用：`cd ../api && buf generate --template buf.servora.typescript.gen.yaml`
- 仓库内现有模板文件：`../api/buf.typescript.gen.yaml`
- 生成结果：`src/service/gen/`
- 开发代理：`vite.config.ts` 把 `/api` 转发到 `http://127.0.0.1:8000`

## 常用命令

```bash
bun install
bun dev
bun run build
bun test:unit
bun test:e2e
bun lint
bun format
bun run gen:proto
```

## 当前脚本事实

- `build` 实际是 `run-p type-check "build-only {@}" --`
- 类型检查命令是 `vue-tsc --build`
- `lint` 会执行 `eslint . --fix --cache`
- `gen:proto` 脚本引用的模板名与仓库当前实际文件名不一致；若执行失败，优先核对 `package.json` 与 `../api/` 下模板命名
- 目录里同时存在 `pnpm-lock.yaml` 与 `pnpm-workspace.yaml`，但项目文档默认命令仍以 Bun 为主

## 维护提示

- 所有旧文档里指向根目录 `web/` 的路径都应改成当前目录
- 这里的自动生成文件包括 `src/auto-imports.d.ts`、`src/components.d.ts`、`src/service/gen/`
- 若修改前端依赖协议，先更新服务 proto，再执行 `bun run gen:proto`
