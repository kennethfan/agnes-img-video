# UI Redesign — Agnes Creator Studio

## Overview

Complete visual refresh of the frontend SPA (Vue 3 + Element Plus + TypeScript). Primary goals: replace dense tab-bar navigation with a modern sidebar pattern, replace bare-bones form layouts with left-input/right-preview dual columns, and apply a minimal/whitespace-heavy visual style.

## Navigation

Replace the current 11-tab `el-tabs` strip with a **collapsed left icon bar + flyout group menu**.

- **Icons bar**: fixed 52px-width column on the left edge. Each icon represents a functional group (图片, 视频, 工具, 作品). Active group is highlighted with solid black background + white icon; inactive groups use light grey background.
- **Flyout groups**: clicking an icon opens a floating card to its right containing the individual page entries for that group. The flyout has the same border-radius and shadow as content cards.
- **Grouping scheme**:
  - 🖼 图片: 文生图, 图生图, 批量生成
  - 🎬 视频: 文生视频, 图生视频, 多图视频
  - 📝 工具: 脚本生成, 点子库, 分镜
  - 🖥 作品: 作品库, 历史记录
- The icon bar sits below a thin top header bar (see Visual Style).

The full-content top tab bar is removed. This also eliminates the `el-tabs` wrapper around page views.

## Visual Style

**Minimal / whitespace-heavy — Linear / Vercel inspired.**

- **Background**: `#ffffff` (pure white) for content areas, `#fafafa` for subtle distinction (e.g., preview panels).
- **Surfaces**: white cards with `1px solid #eaeaea` borders, `border-radius: 12px`.
- **Top header bar**: very thin (`1px border-bottom solid #f0f0f0`), just the studio name left-aligned and optional right-area (search, avatar). No heavy branding.
- **Typography**: ElPlus default (PingFang SC / Inter) but weights kept light (`400` for body, `500` for subheadings, `600` for titles).
- **Primary color**: black `#000000` for primary actions (buttons, active states) — this is a deliberate departure from Element Plus's default blue to match the minimalist brand. Secondary actions use `#f5f5f5` backgrounds.
- **Spacing**: generous padding — `20px` card padding, `16px` between sections. The current UI feels cramped by comparison.
- **Scoped scoped style overrides** in each view use the new card pattern. No global CSS variable change to Element Plus component internals unless unavoidable.

## Page Layout

Every "generation" page (文生图, 图生图, 批量生成, 文生视频, 图生视频, 多图视频) follows the same **dual-column** structure:

- **Left column (input)**: `flex: 1`, contains the form controls (prompt, options, generate button). Separated from the right by a `1px solid #f0f0f0` divider.
- **Right column (preview)**: `flex: 1`, light grey `#fafafa` background initially showing a placeholder, then displays the generated result (images or video). For video pages the preview area includes the SSE progress indicator.

Other pages adapt this pattern:
- **Tools pages (脚本生成, 点子库, 分镜)**: predominantly single-column with card-grouped sections, consistent card styling.
- **作品库 / 历史记录**: grid layout (as now), but cards updated to the new border/radius/shadow pattern.

## Color Palette

```
--bg-page:        #ffffff
--bg-subtle:      #fafafa
--bg-card:        #ffffff
--border-default: #eaeaea
--border-light:   #f0f0f0
--text-primary:   #000000
--text-secondary: #666666
--text-muted:     #909399
--accent:         #000000
--accent-hover:   #333333
--tag-bg:         #f5f5f5
```

## Component Updates

### App.vue
- Remove `<el-tabs>` wrapper. Replace with a top header bar + left icon bar + routed content area.
- The left icon bar and flyout group menu replace the tab-switching logic. Active group/page state managed via a single `activeGroup` and `activePage` ref.

### ShotCard.vue / AssetCard.vue / ImageResult.vue
- Card borders updated to `#eaeaea`, radius to `12px`, remove internal box-shadows.
- Hover state: subtle border color change only, no shadow.

### Buttons
- Primary: black `#000000` background, white text, `border-radius: 8px`, no border.
- Plain/secondary: `#f5f5f5` background, dark text, same radius.
- Text buttons: no background, default text weight.

## Implementation Order

1. **App.vue navigation restructure** — replace el-tabs with icon bar + flyout menu
2. **Visual style CSS** — apply new colors, border, spacing, card patterns
3. **Dual-column layout** — refactor generation pages to left-input/right-preview
4. **Component polish** — update ShotCard, AssetCard, ImageResult, buttons
5. **QA pass** — verify all 10 pages render correctly, no broken states

## Non-Goals

- No change to backend API structure.
- No addition of new features or pages.
- No vue-router migration (still managed via activePage ref + conditional rendering).
- No dark mode support in this iteration.
