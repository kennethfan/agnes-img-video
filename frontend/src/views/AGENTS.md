# frontend/src/views/ — UI Pages

**10 views + 3 wizard subdirs** (comic/novel/image). Navigation via `el-tabs` in `App.vue`, not Vue Router.

## VIEW INDEX

| View | Route Tab | Purpose |
|------|-----------|---------|
| `TextToImage.vue` | 文生图 | Single prompt → image |
| `ImageToImage.vue` | 图生图 | Upload/URL + prompt → refines image |
| `BatchGen.vue` | 批量生成 | Multiple prompts → parallel images |
| `TextToVideo.vue` | 文生视频 | Prompt → video with SSE progress |
| `ImageToVideo.vue` | 图生视频 | Image upload/URL + prompt → video |
| `MultiImageVideo.vue` | 多图视频 | Multiple URLs + keyframes mode |
| `ScriptGen.vue` | 脚本生成 | Topic → AI script (zh/en) |
| `Ideas.vue` | 点子库 | Idea expansion templates |
| `History.vue` | 历史记录 | CRUD + redo to any view |
| `Storyboard.vue` | 分镜 | Projects + shots CRUD + batch generate |

## WIZARD SUBDIRS (comic/novel/image)

Three step-based wizards, all driven by `useWizardStore` in `stores/wizard.ts`:

| Subdir | Steps | Key State |
|--------|-------|-----------|
| `comic/` | Theme → Layout → Prompts → Export | `comic.theme`, `.layout`, `.panels[]` |
| `novel/` | Theme → Genre → Chapters → Export | `novel.theme`, `.genre`, `.chapters[]` |
| `image/` | Upload → Refine → Compare → Export | `image.sourceFile`, `.refinePrompt`, `.strength` |

## CONVENTIONS

- `<script setup>` + Composition API only
- Wizard steps: navigate via `store.nextStep()`, `store.goToStep(n)`, `store.reset()`
- Cross-view redo: store data in `useRedoStore().setRedoData()` → `App.vue` switches tab via `redo-trigger` custom event
- API calls via `src/api/` wrappers, never raw Axios

## ANTI-PATTERNS

- Do NOT add Vue Router guards — navigation is `el-tabs` controlled from `App.vue`
- Do NOT use Options API
- Do NOT import types from Element Plus directly — use `import { ElMessage } from 'element-plus'` only
