# Fonts (iOS)

The Runa design-system typefaces are **committed** in this directory (OFL) and
registered via the `UIAppFonts` array in `../Info.plist`. XcodeGen bundles them as
resources (the target's `sources` includes the `Runa` folder), so after a checkout
just run `xcodegen generate` and build.

## Bundled files

```
ShipporiMincho-Regular.ttf     ShipporiMincho-Medium.ttf
ZenKakuGothicNew-Regular.ttf   ZenKakuGothicNew-Medium.ttf
CormorantGaramond-Regular.ttf  CormorantGaramond-Medium.ttf
```

Only Regular + Medium are shipped per family; emphasis comes from size and colour,
not heavy weights. `Theme/RunaFonts.swift` passes the FAMILY names to
`Font.custom(...)` — "Shippori Mincho", "Zen Kaku Gothic New", "Cormorant Garamond"
— which match the names inside the files (verified with fonttools).

Cormorant Garamond ships upstream as a variable font; the committed Regular/Medium
were instanced (wght 400 / 500) with `fonttools varLib.instancer`.

> The Japanese faces are full JIS fonts (Shippori ~8.6 MB, Zen ~2.3 MB each). If
> the app binary size matters, consider a glyph subset — but that risks tofu for
> uncommon kanji in free-form diary text, so the full faces are shipped by default.
