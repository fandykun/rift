# DESIGN.md — Rift Design System & UI Specification

> This document is the single source of truth for all frontend visual decisions. It is derived from the official mockups provided in the project's design package. Every color value, spacing token, typography scale, and component pattern here must be implemented exactly as specified — no substitutions.

---

## 1. Design Philosophy

Rift's UI is built for **Pro Developers** — users who prioritize speed, precision, and information density over decoration. The aesthetic is **Brutalist-lite + Modern Corporate**: clear hierarchy, structural borders, and a rigid grid. The emotional target is *controlled power* — the interface feels fast because it is visually light despite its functional density.

There are no gradients, no rounded pill buttons on primary actions, no large drop shadows, no marketing language. Every element earns its place by reducing cognitive load during high-stakes operations.

The signature element of Rift is the **split-pane SQL diff viewer** — two CodeMirror panels side-by-side showing `LOCAL MIGRATIONS` vs `LIVE DATABASE`. This is the product's core identity and must be the most polished component.

---

## 2. Color System

All colors come from the `High-Performance DB Tooling` token set. These must be registered as Tailwind custom colors in `tailwind.config.ts`.

### Full Token Map

```ts
colors: {
  // Surfaces — layered from darkest to brightest
  "background":                  "#10131a",  // Page background
  "surface":                     "#10131a",  // Same as background
  "surface-dim":                 "#10131a",
  "surface-container-lowest":    "#0b0e15",  // Deepest inset (code areas)
  "surface-container-low":       "#191b23",  // Table row hover, subtle bg
  "surface-container":           "#1d2027",  // Cards, panels, sidebar
  "surface-container-high":      "#272a31",  // Table headers, active chips
  "surface-container-highest":   "#32353c",  // Modals, elevated elements
  "surface-bright":              "#363941",
  "surface-variant":             "#32353c",  // Hover states

  // Text
  "on-surface":                  "#e1e2ec",  // Primary text
  "on-surface-variant":          "#c2c6d6",  // Secondary text, placeholder
  "on-background":               "#e1e2ec",

  // Borders
  "outline":                     "#8c909f",  // Visible borders
  "outline-variant":             "#424754",  // Subtle borders, dividers

  // Primary — Electric Blue (actions, focus, active nav)
  "primary":                     "#adc6ff",
  "on-primary":                  "#002e6a",
  "primary-container":           "#4d8eff",
  "on-primary-container":        "#00285d",
  "primary-fixed":               "#d8e2ff",
  "primary-fixed-dim":           "#adc6ff",
  "on-primary-fixed":            "#001a42",
  "on-primary-fixed-variant":    "#004395",
  "inverse-primary":             "#005ac2",
  "surface-tint":                "#adc6ff",

  // Secondary — Emerald (success, applied, healthy)
  "secondary":                   "#4edea3",
  "on-secondary":                "#003824",
  "secondary-container":         "#00a572",
  "on-secondary-container":      "#00311f",
  "secondary-fixed":             "#6ffbbe",
  "secondary-fixed-dim":         "#4edea3",
  "on-secondary-fixed":          "#002113",
  "on-secondary-fixed-variant":  "#005236",

  // Tertiary — Amber (warnings, linting, non-breaking issues)
  "tertiary":                    "#ffb95f",
  "on-tertiary":                 "#472a00",
  "tertiary-container":          "#ca8100",
  "on-tertiary-container":       "#3e2400",
  "tertiary-fixed":              "#ffddb8",
  "tertiary-fixed-dim":          "#ffb95f",
  "on-tertiary-fixed":           "#2a1700",
  "on-tertiary-fixed-variant":   "#653e00",

  // Error — Red (failures, critical linter errors, rollbacks)
  "error":                       "#ffb4ab",
  "on-error":                    "#690005",
  "error-container":             "#93000a",
  "on-error-container":          "#ffdad6",

  // Inverse
  "inverse-surface":             "#e1e2ec",
  "inverse-on-surface":          "#2e3038",
}
```

### Semantic Color Usage

| Concept | Token(s) |
|---|---|
| Page background | `background` |
| Sidebar / panels | `surface-container` |
| Table header row | `surface-container-high` |
| Card / panel border | `outline-variant` |
| Primary text | `on-surface` |
| Dimmed / metadata text | `on-surface-variant` |
| Primary action (Apply Migrations) | `bg-primary text-on-primary` |
| Secondary action (Preview Changes) | `border-outline-variant text-on-surface` |
| Applied / success state | `text-secondary`, `bg-secondary-container/20`, `border-secondary/30` |
| Pending state | `text-on-surface-variant`, `bg-surface-container-low`, `border-outline-variant` |
| Error / failed state | `text-error`, `bg-error-container/10`, `border-error/50` |
| Linter warning | `text-tertiary`, `bg-tertiary-container`, `border-tertiary/30` |
| Active nav item | `text-primary bg-primary-container/10 border-r-2 border-primary` |
| SQL keyword | `text-primary` (`#adc6ff`) |
| SQL table name | `text-secondary` (`#4edea3`) |
| SQL data type | `text-tertiary` (`#ffb95f`) |
| SQL string literal | `text-secondary-container` (`#00a572`) |
| SQL comment | `text-outline` italic (`#8c909f`) |
| Diff addition (new line) | `text-secondary` with `bg-secondary/5` row background |
| Diff deletion (removed line) | `text-error` with `bg-error/5` row background and strikethrough |

---

## 3. Typography

### Font Families

| Role | Family | Usage |
|---|---|---|
| Display & Headings | **Geist** | Page titles, section headers, sidebar brand |
| Body & UI | **Inter** | Table cells, descriptions, button labels, form fields |
| Code & Technical Data | **JetBrains Mono** | SQL editors, version IDs, checksums, terminal output, table header labels, status chips |

Load via Google Fonts:
```html
<link href="https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet"/>
```

### Type Scale

```ts
fontSize: {
  "display":      ["32px", { lineHeight: "1.2", letterSpacing: "-0.02em", fontWeight: "600" }],
  "headline-md":  ["24px", { lineHeight: "1.3", fontWeight: "600" }],
  "headline-sm":  ["18px", { lineHeight: "1.4", fontWeight: "500" }],
  "body-lg":      ["16px", { lineHeight: "1.6", fontWeight: "400" }],
  "body-md":      ["14px", { lineHeight: "1.5", fontWeight: "400" }],
  "code-md":      ["14px", { lineHeight: "1.5", fontWeight: "400" }],
  "code-sm":      ["12px", { lineHeight: "1.4", fontWeight: "400" }],
  "label-caps":   ["11px", { lineHeight: "1", letterSpacing: "0.05em", fontWeight: "600" }],
}

fontFamily: {
  "display":      ["Geist", "sans-serif"],
  "headline-md":  ["Geist", "sans-serif"],
  "headline-sm":  ["Geist", "sans-serif"],
  "body-lg":      ["Inter", "sans-serif"],
  "body-md":      ["Inter", "sans-serif"],
  "code-md":      ["JetBrains Mono", "monospace"],
  "code-sm":      ["JetBrains Mono", "monospace"],
  "label-caps":   ["JetBrains Mono", "monospace"],
}
```

**Default body size** is `body-md` (14px Inter). Use `label-caps` (11px JetBrains Mono, uppercase, tracked) for table column headers, status chip text, and metadata labels.

---

## 4. Spacing & Layout

```ts
spacing: {
  "base":           "4px",
  "unit-1":         "0.25rem",   // 4px
  "unit-2":         "0.5rem",    // 8px
  "unit-4":         "1rem",      // 16px
  "unit-6":         "1.5rem",    // 24px
  "unit-8":         "2rem",      // 32px
  "container-max":  "1440px",
  "gutter":         "1rem",      // 16px page gutter
  "sidebar-width":  "240px",
}
```

**Layout model:** Fixed-Fluid hybrid. The sidebar is a fixed 240px. The main content area fills the remaining width fluidly up to 1440px max. The main canvas uses `p-unit-8` (32px) padding.

**Grid:** 4px base grid throughout. Use `unit-2` (8px) for compact component internal padding, `unit-4` (16px) for standard card padding.

---

## 5. Border Radius

```ts
borderRadius: {
  "sm":      "0.125rem",  // 2px — tight elements
  "DEFAULT": "0.25rem",   // 4px — inputs, buttons, chips, table rows
  "lg":      "0.5rem",    // 8px — cards, panels
  "xl":      "0.75rem",   // 12px — modals
  "full":    "9999px",    // status dots, avatar initials
}
```

The dominant radius is `DEFAULT` (4px) — everything feels square and technical. Cards use `rounded-lg` (8px). Only status indicator dots and avatar circles use `rounded-full`.

---

## 6. Elevation & Depth

Depth is created through **tonal layering**, not shadows. Each level uses a progressively lighter surface token plus a 1px border.

| Level | Token | Border | Usage |
|---|---|---|---|
| 0 — Base | `background` (#10131a) | none | Page canvas |
| 1 — Content | `surface-container` (#1d2027) | `border-outline-variant` | Sidebar, cards, panels |
| 2 — Header | `surface-container-high` (#272a31) | `border-outline-variant` | Table headers, tab bars |
| 3 — Elevated | `surface-container-highest` (#32353c) | `border-outline-variant` | Modals, dropdowns |
| 4 — Glassmorphism | `surface-container/80` + `backdrop-blur-sm` | `border-outline-variant/50` | Linter alerts overlay, conflict cards |

**Glassmorphism** (Level 4) is used sparingly — only the Linter Alerts sidebar card on the Dashboard uses it. It has a subtle amber/red inner glow when it contains warnings: `shadow-[inset_0_0_20px_rgba(255,185,95,0.05)]` for warnings, `rgba(173,198,255,0.05)` for pending state inset glow.

---

## 7. Component Library

### 7.1 Sidebar Navigation

Fixed 240px, `bg-surface-container`, `border-r border-outline-variant`.

**Brand header** (top):
```html
<div class="px-unit-4 mb-unit-8 flex items-center gap-unit-2">
  <div class="w-8 h-8 rounded bg-primary-container flex items-center justify-center text-on-primary-container">
    <!-- database icon (Material Symbols, FILL=1) -->
  </div>
  <div>
    <h1 class="font-display text-headline-sm font-bold text-primary">Rift DB</h1>
    <p class="font-label-caps text-label-caps text-on-surface-variant uppercase">PostgreSQL Instance</p>
  </div>
</div>
```

**Nav item — inactive:**
```html
<a class="flex items-center gap-unit-2 px-unit-2 py-2 rounded-DEFAULT text-on-surface-variant hover:bg-surface-variant transition-colors cursor-pointer active:scale-95">
  <!-- icon 20px --> <span>Label</span>
</a>
```

**Nav item — active:**
```html
<a class="flex items-center gap-unit-2 px-unit-2 py-2 rounded-DEFAULT text-primary bg-primary-container/10 border-r-2 border-primary cursor-pointer active:scale-95">
  <!-- icon 20px fill --> <span class="font-medium">Label</span>
</a>
```

**CTA Button (bottom of sidebar):**
```html
<button class="w-full bg-primary text-on-primary hover:bg-primary/90 font-body-md text-body-md font-medium py-2 rounded-DEFAULT transition-colors flex items-center justify-center gap-2">
  <!-- add icon 18px --> New Migration
</button>
```

Footer nav items (Docs, Support) live below a `border-t border-outline-variant` divider.

### 7.2 Top App Bar

Sticky, `bg-background border-b border-outline-variant h-14 px-gutter`.

Left: `Docs`, `API`, `Changelog` links — `font-label-caps text-label-caps text-on-surface-variant hover:text-on-surface uppercase`.

Right actions (in order):
1. **"Preview Changes"** — `px-3 py-1.5 border border-outline-variant text-on-surface hover:bg-surface-variant rounded-DEFAULT transition-colors uppercase` (ghost secondary)
2. **"Apply Migrations"** — `px-3 py-1.5 bg-primary text-on-primary hover:bg-primary/90 rounded-DEFAULT font-bold uppercase shadow-sm` (solid primary)
3. Notification bell icon + Account circle icon — `text-on-surface-variant hover:text-on-surface`, separated by `border-l border-outline-variant`

### 7.3 Status Chips

All chips use `font-label-caps text-label-caps uppercase` and `rounded-full` (pill shape). They include a 6px status dot on the left.

**Applied (Success):**
```html
<div class="inline-flex items-center gap-1.5 border border-secondary/30 text-secondary px-2 py-0.5 rounded-full font-label-caps text-label-caps uppercase bg-secondary-container/10">
  <div class="w-1.5 h-1.5 rounded-full bg-secondary"></div> Applied
</div>
```

**Pending (Neutral):**
```html
<div class="inline-flex items-center gap-1.5 border border-outline-variant text-on-surface-variant px-2 py-0.5 rounded-full font-label-caps text-label-caps uppercase bg-surface-container-low">
  <div class="w-1.5 h-1.5 rounded-full bg-outline-variant"></div> Pending
</div>
```

**Pending — Dangerous (has linter error):**
```html
<div class="inline-flex items-center gap-1.5 border border-error/50 text-error px-2 py-0.5 rounded-full font-label-caps text-label-caps uppercase bg-error-container/10">
  <!-- warning triangle icon 14px --> Pending
</div>
```

**Failed:**
```html
<div class="inline-flex items-center gap-1.5 border border-error/50 text-error px-2 py-0.5 rounded-full font-label-caps text-label-caps uppercase bg-error-container/10">
  <!-- error icon 14px --> Failed
</div>
```

**Rolled Back:**
```html
<div class="inline-flex items-center gap-1.5 border border-outline-variant text-on-surface-variant px-2 py-0.5 rounded-full font-label-caps text-label-caps uppercase bg-surface-container-low opacity-60">
  <div class="w-1.5 h-1.5 rounded-full bg-outline-variant"></div> Rolled Back
</div>
```

### 7.4 Stat Cards

Used in the Dashboard overview row (3-column grid). Each card is `bg-surface-container border border-outline-variant p-unit-4 rounded-DEFAULT flex flex-col gap-2 relative overflow-hidden group`.

- Label: `font-label-caps text-label-caps text-on-surface-variant uppercase`
- Value: `font-headline-md text-headline-md text-on-background`
- Ghost icon (top-right, decorative): `absolute top-0 right-0 p-unit-4 text-4xl opacity-20 group-hover:opacity-40 transition-opacity`

**Pending card** gets an elevated treatment: `border-primary/30` border, `shadow-[inset_0_0_20px_rgba(173,198,255,0.05)]` inner glow, and the ghost icon uses `animate-pulse`.

### 7.5 Data Table

**Container:** `bg-surface-container border border-outline-variant rounded-DEFAULT overflow-hidden`

**Header row:** `bg-surface-container-high border-b border-outline-variant font-label-caps text-label-caps text-on-surface-variant uppercase tracking-wider`

Header cells: `p-unit-4 font-semibold`

**Body rows:** `font-body-md text-body-md text-on-surface border-b border-outline-variant hover:bg-surface-variant/50 transition-colors group`

**Version/ID column:** `font-code-sm text-code-sm text-primary` — always monospace, always electric blue.

**Name column:** `font-code-sm text-code-sm text-on-surface` — monospace but not colored.

**Author, Date columns:** `font-body-md text-body-md text-on-surface-variant`

**Actions column:** icon button `px-2 py-1 text-xs border border-error/50 text-error rounded hover:bg-error/10 transition-colors` for Rollback; plain icon for View.

### 7.6 Quick Actions Card

Right sidebar card. Uses ghost/outline button style for secondary actions:

```html
<!-- Primary action -->
<button class="w-full bg-primary text-on-primary ... flex items-center justify-between group">
  Connect to DB
  <span class="w-1.5 h-1.5 rounded-full bg-primary animate-pulse hidden group-hover:block"></span>
</button>

<!-- Secondary action -->
<button class="w-full border border-primary text-primary hover:bg-primary/10 ... flex items-center justify-between group">
  Sync Local Files
</button>
```

### 7.7 Linter Alerts Card (Dashboard sidebar)

Glassmorphism panel: `bg-surface-container/80 backdrop-blur-sm border border-outline-variant rounded-lg`.

Section header shows a pulsing red dot + "1 Issue" count in `text-error`.

Each alert item: `bg-surface-container-high border border-error/50 rounded p-3 relative overflow-hidden`. Left accent bar: `absolute top-0 left-0 w-1 h-full bg-error`. 

Content: `text-error` for the pattern name (bold), `text-on-surface-variant` for description. Inline `<code>` elements use `bg-surface-variant px-1 rounded text-error`.

Action button: `bg-error/10 hover:bg-error/20 text-error font-label-caps px-3 py-1.5 rounded transition-colors border border-error/30`.

---

## 8. Page-by-Page Specifications

### 8.1 Migration Dashboard (`/migrations`)

**Reference mockup:** `rift_dashboard/screen.png`

**Layout:** Full-width bento grid. Top row: 3 stat cards spanning all 12 columns. Below: 8-column main area (migration table) + 4-column right sidebar (Quick Actions + Linter Alerts).

**Page header:**
- Title: `font-display text-display text-on-background` → "Migration Dashboard"
- Subtitle: `font-body-md text-body-md text-on-surface-variant` → "Manage and track database schema evolution."

**"Recent Activity" table section:**
- Section title `font-headline-sm text-headline-sm` + search input right-aligned
- Search input: `bg-surface-container border border-outline-variant rounded-DEFAULT py-1.5 pl-9 pr-3` with a search icon absolutely positioned inside

**Table columns:** Status | ID (Timestamp) | Name | Author | Applied Date | Actions

**Status chip rules (per row):**
- Pending (clean) → neutral chip
- Pending (has linter warning) → error-tinted chip with warning icon
- Applied → success chip
- Failed → error chip

**Sidebar panels (right column):**
1. **Quick Actions** card — "Connect to DB" (primary) + "Sync Local Files" (ghost)
2. **Linter Alerts** card — shows the most recent dangerous pending migration with a "REVIEW" link

### 8.2 Schema Diff Viewer (`/migrations/:version/diff`)

**Reference mockup:** `schema_diff_viewer/screen.png`

**Layout:** Full-height workspace. No page padding — the diff panes fill the viewport. The workspace is structured as:
1. **Summary & Controls Panel** — horizontal bar above the split panes
2. **Split-Pane Diff Viewer** — takes all remaining height

**Summary Panel:**
```
[diff icon]  Schema Summary                    [toggle] SAFE PREVIEW (DDL)
  [add icon] 3 tables to be created  •  [edit icon] 1 column to be altered
```
- Summary panel: `bg-surface-container border-b border-outline-variant p-unit-4 flex items-center justify-between`
- Change counts use `text-secondary` for additions and `text-tertiary` for modifications
- "Safe Preview (DDL)" toggle: a custom toggle (`peer` Tailwind pattern) showing DDL-safe mode

**Split Panes:**

Two panes side-by-side with a 1px `border-r border-outline-variant` divider between them.

Left pane header: `bg-surface-container-high border-b border-outline-variant px-unit-4 py-2 flex items-center justify-between`
- Left label: `● LOCAL MIGRATIONS` in `font-label-caps text-label-caps text-on-surface-variant uppercase` with a pulsing blue dot
- Right label: filename in `font-code-sm text-code-sm text-primary`

Right pane header: same structure but
- Left label: `○ LIVE DATABASE` 
- Right label: `schema: public` in `font-code-sm text-code-sm text-on-surface-variant`

**SQL rendering in panes:** Uses syntax-highlighted monospace rendering. Each line has a line number column (`text-on-surface-variant/40 select-none w-8 text-right pr-4 font-code-sm`).

**Diff line types:**

| Type | Row background | Text treatment |
|---|---|---|
| Unchanged | transparent | `text-on-surface` |
| Addition (new) | `bg-secondary/5` | `text-secondary` |
| Deletion (removed) | `bg-error/5` | `text-error line-through opacity-70` |
| Missing/comment | transparent | `text-on-surface-variant italic` (e.g. `/* index idx_users_email missing */`) |

**SQL syntax colors** (consistent across all code views):
- Keywords (`CREATE TABLE`, `ALTER TABLE`, `PRIMARY KEY`, etc.): `text-primary` (`#adc6ff`)
- Table/index names: `text-secondary` (`#4edea3`)
- Data types (`UUID`, `VARCHAR`, `TIMESTAMP`): `text-tertiary` (`#ffb95f`)
- String literals: `text-secondary-container` (`#00a572`)
- Comments: `text-outline italic` (`#8c909f`)
- Operators/punctuation: `text-on-surface-variant` (`#c2c6d6`)

**Top-right "APPLY MIGRATIONS" button** on this page: solid primary, `font-label-caps text-label-caps uppercase font-bold`. Clicking opens a confirmation modal (glassmorphism Level 4, `backdrop-blur-sm`).

### 8.3 SQL Authoring Interface (`/migrations/:version` — edit mode)

**Reference mockup:** `sql_authoring_interface/screen.png`

**Layout:** Three-panel horizontal split:
1. **Left panel (240px fixed):** Available Schemas / Table Browser
2. **Center panel (fluid):** SQL Editor
3. **Right panel (280px fixed):** Zero-Downtime Linter

**Left Panel — Table Browser:**
- Header: "Available Schemas" in `font-headline-sm text-headline-sm`
- Filter input: `bg-surface-container border border-outline-variant rounded py-1 pl-8 font-code-sm` with search icon
- Schema tree: schema label (`public`, `auth`) as collapsible folders, table rows as `flex items-center gap-2 py-1 px-2 rounded hover:bg-surface-variant/50 cursor-pointer font-code-sm text-code-sm`
- Table icon: a small grid icon 14px `text-primary`
- Clicking a table name inserts it into the active cursor position in the editor

**Center Panel — SQL Editor:**
- Metadata strip at top: filename input (editable, `font-code-md`), Category dropdown, Author badge
- Metadata strip: `bg-surface-container-high border-b border-outline-variant px-unit-4 py-2 flex items-center gap-unit-4`
- Filename input: `bg-transparent border-none focus:ring-0 font-code-md text-code-md text-on-surface` 
- Category dropdown: `bg-surface-container-high border border-outline-variant rounded text-sm font-body-md`
- Author badge: avatar circle (colored by name hash) + name label, `font-code-sm text-code-sm`
- Editor area: `bg-surface-container-lowest` (darkest surface), line numbers column left, code area right
- **Linter error line** highlight: `bg-error-container/10 border-l-2 border-error -ml-4 pl-4` on the offending line
- Error line number background: `text-error font-bold bg-error-container/20`

**Right Panel — Zero-Downtime Linter:**
- Panel header: "Zero-Downtime Linter" title + pulsing red dot + "1 Issue" count badge
- Status card (error): `bg-surface-container-high border border-error/50 rounded p-3 relative overflow-hidden` with left `w-1 h-full bg-error` accent bar
  - Icon: Material Symbols `error` in `text-error mt-0.5`
  - Message with inline `<code>` elements: `bg-surface-variant px-1 rounded text-error`
  - Auto-Fix button: `bg-error/10 hover:bg-error/20 text-error font-label-caps px-3 py-1.5 rounded border border-error/30`
- Divider: `border-t border-outline-variant my-3`
- Passed rules: `flex items-center gap-2 text-on-surface-variant font-body-md text-body-md`
  - Green dot: `w-1.5 h-1.5 rounded-full bg-secondary`
  - Rule name + pass label

### 8.4 Team & Deployment History (`/team`)

**Reference mockup:** `team_deployment_history/screen.png`

**Layout:** Two-column: 8-column left (Conflict Detection + Deployment History Timeline) + 4-column right (Team Access + Notifications).

**Conflict Detection Card:**
- Container: `bg-surface-container rounded border border-error shadow-sm overflow-hidden`
- Header: `flex items-center justify-between p-unit-4` with "Conflict Detection" title in `text-error` + "1 CONFLICT DETECTED" badge (`bg-error/20 text-error border border-error/30 font-label-caps uppercase rounded px-2 py-0.5`)
- Conflict body shows the two conflicting branches side by side as mini code cards
- Each branch card: monospace code snippet of the conflicting DDL, branch name label above
- "Resolve manually" CTA button: secondary style

**Deployment History Timeline:**
- Section title: "Deployment History" + "View All →" link
- Timeline uses a vertical line `border-l-2 border-outline-variant ml-2.5 pl-6` structure
- Each timeline item has an absolute circle marker: `absolute w-3 h-3 bg-secondary rounded-full -left-[6.5px] top-1.5 ring-4 ring-surface-container` (success) or `bg-error` (failure)
- Success item: migration name in monospace + `SUCCESS` chip + "Applied to Production via CI/CD" subtitle + author avatar + "Deployed by Name"
- Failed item: migration name + `FAILED` chip + "Rollback triggered automatically" + inline error message in `bg-surface-container-low border border-error/30 rounded font-code-sm text-error p-2`

**Team Access Panel (right):**
- Section header: "Team Access" + `+` icon button
- Each member row: Avatar (colored initials circle, 36px) + Name/email + Role badge
- Avatar: 36px circle, colored per role — Admin: `bg-primary-container text-on-primary-container`, Developer: `bg-tertiary-container text-on-tertiary-container`, Viewer: `bg-surface-variant text-on-surface-variant`
- Role badge: plain text chip `font-label-caps uppercase text-on-surface-variant border border-outline-variant rounded px-2 py-0.5`

**Notifications / Webhooks Panel (right, below Team):**
- Toggle rows for Slack and Discord integration
- Toggle: `w-10 h-5 bg-surface-variant rounded-full peer peer-checked:bg-primary/30 border border-outline-variant peer-checked:border-primary/50` with inner thumb
- Webhook URL input below each active toggle

---

## 9. SQL Syntax Highlighting Reference

Applied in both the editor (CodeMirror 6 theme) and the read-only diff panes. Use the following CSS custom properties or CodeMirror style override:

```css
/* Inject as a CodeMirror theme or inline span classes */
.sql-keyword  { color: #adc6ff; font-weight: 600; }   /* primary */
.sql-table    { color: #4edea3; }                       /* secondary */
.sql-type     { color: #ffb95f; }                       /* tertiary */
.sql-string   { color: #00a572; }                       /* secondary-container */
.sql-comment  { color: #8c909f; font-style: italic; }  /* outline */
.sql-operator { color: #c2c6d6; }                       /* on-surface-variant */
.sql-number   { color: #ffb95f; }                       /* tertiary */
```

For CodeMirror 6, register these as a `HighlightStyle` using `@codemirror/language` tags. The editor background must be `surface-container-lowest` (#0b0e15).

---

## 10. Icons

Use **Material Symbols Outlined** throughout (variable font, loaded from Google Fonts). All icons are 20px in nav, 18px in buttons, 14px in chips, 24px in stat card decorators.

Key icon mappings:
| Usage | Icon name |
|---|---|
| Sidebar brand | `database` (FILL=1) |
| Dashboard nav | `dashboard` |
| Migrations nav | `database_off` |
| Schema Diff nav | `difference` |
| Team nav | `group` |
| Settings nav | `settings` |
| New Migration CTA | `add` |
| Docs footer | `description` |
| Support footer | `help` |
| Notifications | `notifications` |
| Account | `account_circle` |
| Search | `search` |
| Linter error | `error` |
| Conflict warning | `warning` (amber) |
| Success check | `check_circle` |
| Pending | `pending_actions` |
| Diff/compare | `difference` |
| Quick actions | `bolt` |

---

## 11. Tailwind Configuration

The complete `tailwind.config.ts` structure:

```ts
import type { Config } from 'tailwindcss'

export default {
  darkMode: 'class',  // always dark; set <html class="dark">
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: { /* full token map from Section 2 */ },
      borderRadius: {
        'sm': '0.125rem',
        'DEFAULT': '0.25rem',
        'lg': '0.5rem',
        'xl': '0.75rem',
        'full': '9999px',
      },
      spacing: {
        'base': '4px',
        'unit-1': '0.25rem',
        'unit-2': '0.5rem',
        'unit-4': '1rem',
        'unit-6': '1.5rem',
        'unit-8': '2rem',
        'container-max': '1440px',
        'gutter': '1rem',
        'sidebar-width': '240px',
      },
      fontFamily: { /* from Section 3 */ },
      fontSize: { /* from Section 3 */ },
    }
  },
  plugins: [require('@tailwindcss/forms'), require('@tailwindcss/container-queries')],
} satisfies Config
```

---

## 12. Navigation Structure

The sidebar nav has exactly 5 main items + 2 footer items:

**Main nav (top):**
1. Dashboard (`/`) — `dashboard` icon
2. Migrations (`/migrations`) — `database_off` icon
3. Schema Diff (`/migrations/:version/diff`) — `difference` icon
4. Team (`/team`) — `group` icon
5. Settings (`/settings`) — `settings` icon

**Footer nav (below divider):**
6. Docs (external link)
7. Support (external link)

**CTA button** ("+ New Migration") lives just above the footer nav at the bottom of the sidebar.

The active nav item uses a **right-side 2px primary border** accent (`border-r-2 border-primary`) — not a background pill. This is the signature nav pattern.

---

## 13. Responsive Behavior

Desktop (≥1280px): full layout with sidebar + content.

Tablet (768–1279px): sidebar collapses to icon-only (48px wide), tooltip on hover reveals labels. Table truncates Author column.

Mobile (<768px): sidebar becomes a bottom drawer triggered by a hamburger. Page padding reduces to 16px. Stat cards stack vertically. Diff viewer stacks panes vertically (top = Local, bottom = Live).

---

## 14. Animations & Motion

Keep motion minimal and purposeful:
- `transition-colors duration-200 ease-in-out` on all interactive hover states
- `active:scale-95` on buttons (tactile press feedback)
- `animate-pulse` on: pending stat card ghost icon, linter error dot, conflict detection badge
- `transition-transform` on sidebar toggle thumb
- No page transition animations — route changes are instant
- No entrance animations on list items — data loads in place

The deploy log page (live SSE stream) auto-scrolls to the bottom as new log lines arrive. Each new line fades in with `animate-[fadeIn_0.15s_ease-in]`.

---

## 15. Empty & Loading States

**Loading skeleton:** `bg-surface-container-high animate-pulse rounded-DEFAULT` blocks. Table rows use 3 skeleton lines per row. Stat cards show a single wide skeleton block.

**Empty state (no migrations):**
```
[database icon, 48px, text-on-surface-variant/30]
No migrations yet
[body-md text-on-surface-variant]
Run `rift new <name>` to create your first migration file.
[code-sm text-primary mt-1]
```
Centered in the table area, no borders, no card.

**Error state:** Red bordered card with `error` icon + "Something went wrong" + error message in code-sm + "Retry" ghost button.

---

## 16. Migration from PRD Page Specs

The following PRD page specs are superseded by the mockup-derived specs in Section 8 of this document:

| PRD Section | Superseded By |
|---|---|
| "Web Dashboard — Routes" layout descriptions | Section 8 of this document |
| StatusBadge component description | Section 7.3 |
| Dashboard stat cards | Section 7.4 |
| Diff viewer description | Section 8.2 |
| History page layout | Section 8.4 |

The PRD's API endpoints, data models, Go architecture, and CLI specs remain the authoritative source for backend implementation. This document is authoritative for all frontend visual decisions.