# Fonts

The Runa design system uses three custom families. Font binaries are **not**
committed to this repo — download them from Google Fonts and drop the files
into **this directory** (`apps/swift/Runa/Fonts/`).

| Role     | Family (used in `Font.custom`) | Google Fonts page                                    |
| -------- | ------------------------------ | ---------------------------------------------------- |
| Headings | `Shippori Mincho`              | https://fonts.google.com/specimen/Shippori+Mincho    |
| Body     | `Zen Kaku Gothic New`          | https://fonts.google.com/specimen/Zen+Kaku+Gothic+New|
| Logo     | `Cormorant Garamond`           | https://fonts.google.com/specimen/Cormorant+Garamond |

## Expected filenames

Place exactly these files here (these are the names referenced by the commented
`UIAppFonts` block in `../Info.plist`). Add or remove weights as needed, but keep
`Info.plist` in sync:

```
ShipporiMincho-Regular.ttf
ShipporiMincho-Medium.ttf
ShipporiMincho-SemiBold.ttf
ZenKakuGothicNew-Regular.ttf
ZenKakuGothicNew-Medium.ttf
CormorantGaramond-Regular.ttf
CormorantGaramond-Medium.ttf
```

## After adding the files

1. Uncomment the `UIAppFonts` array in `apps/swift/Runa/Info.plist`.
2. Re-run `xcodegen generate` (so the files are picked up as bundle resources).
3. Build & run. Verify the family name reported by iOS matches the string passed
   to `Font.custom(...)` in `Theme/RunaFonts.swift` — the **PostScript / family
   name inside the file** is what must match, not the filename. If text still
   renders in the system font, log the available names once to check:

   ```swift
   for family in UIFont.familyNames.sorted() {
       print(family, UIFont.fontNames(forFamilyName: family))
   }
   ```

Until the files are present, `Font.custom` falls back to the system font
gracefully, so the app builds and runs without them.
