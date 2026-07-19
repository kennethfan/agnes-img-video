# Agnes Creator Studio — Frontend

Vue 3 + TypeScript 6 + Vite 8 + Element Plus 前端应用。

## 快速开始

```bash
pnpm install
pnpm dev          # 开发模式 :5173，代理 /api + /outputs → :8080
pnpm build        # 类型检查 + 生产构建
```

## 目录结构

```
src/
├── views/                  # 20+ 页面组件
│   └── comic/              # 漫画创建向导
├── components/             # 复用组件（ImageResult、ShotCard、TaskProgress 等）
├── api/                    # 17 个 Axios API 封装
├── stores/                 # Pinia 状态管理
├── types/                  # TypeScript 类型定义
└── utils/                  # SSE、工具函数
```

## Vite 代理

- `/api` → `http://localhost:8080`
- `/outputs` → `http://localhost:8080`
