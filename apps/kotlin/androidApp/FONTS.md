# Fonts (Android)

Font binaries are NOT committed. The theme falls back to the system default
(`FontFamily.Default`) so the build stays green without them.

## Enabling the real Runa typefaces

1. Download from Google Fonts:
   - Shippori Mincho — https://fonts.google.com/specimen/Shippori+Mincho
   - Zen Kaku Gothic New — https://fonts.google.com/specimen/Zen+Kaku+Gothic+New
   - Cormorant Garamond — https://fonts.google.com/specimen/Cormorant+Garamond

2. Create `androidApp/src/main/res/font/` and drop the `.ttf` files there.
   Android resource filenames must be lowercase with underscores only. Use these
   EXACT names — the (commented-out) wiring in `ui/theme/Type.kt` expects them:

   | Role     | Family              | Filename                          |
   |----------|---------------------|-----------------------------------|
   | headings | Shippori Mincho     | `shippori_mincho_regular.ttf`     |
   | headings | Shippori Mincho     | `shippori_mincho_bold.ttf`        |
   | body     | Zen Kaku Gothic New | `zen_kaku_gothic_new_regular.ttf` |
   | body     | Zen Kaku Gothic New | `zen_kaku_gothic_new_medium.ttf`  |
   | logo     | Cormorant Garamond  | `cormorant_garamond_regular.ttf`  |

3. In `ui/theme/Type.kt`, uncomment the `FontFamily(...)` blocks and change the
   three `val ... = FontFamily.Default` assignments to the uncommented families.

> A doc file can NOT live under `res/` — Android's resource merger rejects any
> filename that is not `.xml/.ttf/.ttc/.otf`. That is why this note lives at the
> module root instead of inside `res/font/`.
