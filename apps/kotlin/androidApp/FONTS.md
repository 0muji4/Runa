# Fonts (Android)

The Runa design-system typefaces are **committed** under
`androidApp/src/main/res/font/` (OFL — Open Font License) and wired in
`ui/theme/Type.kt`. Only Regular + Medium are shipped per family; emphasis comes
from size and colour, not heavy weights, in keeping with the quiet tone.

| Role     | Family              | Files                                                            |
|----------|---------------------|-----------------------------------------------------------------|
| headings | Shippori Mincho     | `shippori_mincho_regular.ttf`, `shippori_mincho_medium.ttf`     |
| body     | Zen Kaku Gothic New | `zen_kaku_gothic_new_regular.ttf`, `zen_kaku_gothic_new_medium.ttf` |
| logo     | Cormorant Garamond  | `cormorant_garamond_regular.ttf`, `cormorant_garamond_medium.ttf` |

Sources (Google Fonts): [Shippori Mincho](https://fonts.google.com/specimen/Shippori+Mincho),
[Zen Kaku Gothic New](https://fonts.google.com/specimen/Zen+Kaku+Gothic+New),
[Cormorant Garamond](https://fonts.google.com/specimen/Cormorant+Garamond).
Cormorant Garamond ships upstream as a variable font; the committed Regular/Medium
were instanced from it (wght 400 / 500) with `fonttools varLib.instancer`.

> Note: the Japanese faces are full JIS fonts (Shippori ~8.6 MB each, Zen ~2.3 MB
> each), so `res/font/` is ~22 MB. If repo size becomes a concern, consider Git LFS
> or a glyph subset — but a subset risks tofu for uncommon kanji in free-form diary
> text, so the full faces are shipped by default.
