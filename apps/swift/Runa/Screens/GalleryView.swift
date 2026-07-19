import PhotosUI
import Shared
import SwiftUI

/// 13 ギャラリー — "ひかりの記録". A whitespace-rich two-column masonry of the user's
/// images with a gallery-scoped display-theme toggle (monotone ⇔ pink) that re-grades
/// the whole grid — NOT the app-wide theme. Tapping a cell opens the lightbox (14).
/// Everything renders from the local DB; adds queue offline and flush on reconnect.
/// Matches the Android GalleryScreen so both OSes agree.
struct GalleryView: View {
    @StateObject private var model = GalleryObservable()
    @State private var pickerItem: PhotosPickerItem?
    @State private var lightbox: LightboxContext?

    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()
            VStack(spacing: 0) {
                header
                content
            }
        }
        .onChange(of: pickerItem) { newItem in loadPicked(newItem) }
        .fullScreenCover(item: $lightbox) { ctx in
            LightboxView(context: ctx) { clientId in model.deleteImage(clientId: clientId) }
        }
    }

    // MARK: header + toggle

    private var header: some View {
        ZStack {
            Text("ひかりの記録")
                .font(RunaFonts.heading(26))
                .tracking(6)
                .foregroundStyle(RunaColors.heading)
            HStack {
                Spacer()
                PhotosPicker(selection: $pickerItem, matching: .images) {
                    Text("＋").font(RunaFonts.body(24)).foregroundStyle(RunaColors.subtle)
                }
            }
        }
        .padding(.top, 20)
        .padding(.horizontal, 20)
    }

    @ViewBuilder private var content: some View {
        if let state = model.state {
            switch onEnum(of: state) {
            case .content(let c):
                themeToggle(c.displayTheme)
                bannerLabel(c.banner)
                grid(images: c.images, theme: c.displayTheme)
            case .empty(let e):
                themeToggle(e.displayTheme)
                bannerLabel(e.banner)
                emptyState()
            case .loading:
                Spacer()
            case .error:
                emptyState()
            }
        } else {
            Spacer()
        }
    }

    private func themeToggle(_ selected: GalleryDisplayTheme) -> some View {
        HStack {
            Spacer()
            HStack(spacing: 4) {
                themeSegment("モノトーン", selected: isMonotone(selected)) { model.setDisplayTheme(.monotone) }
                themeSegment("ピンク", selected: !isMonotone(selected)) { model.setDisplayTheme(.pink) }
            }
            .padding(4)
            .background(RunaColors.surface)
            .clipShape(Capsule())
            Spacer()
        }
        .padding(.top, 8)
    }

    private func themeSegment(_ label: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(label)
                .font(RunaFonts.body(15))
                .foregroundStyle(selected ? RunaColors.background : RunaColors.subtle)
                .padding(.horizontal, 26)
                .padding(.vertical, 8)
                .background(selected ? RunaColors.accent : Color.clear)
                .clipShape(Capsule())
        }
        .buttonStyle(.plain)
    }

    // MARK: grid (two-column masonry)

    private func grid(images: [GalleryImage], theme: GalleryDisplayTheme) -> some View {
        let split = masonry(images)
        return ScrollView {
            HStack(alignment: .top, spacing: 16) {
                column(split.0, allImages: images, theme: theme)
                column(split.1, allImages: images, theme: theme)
            }
            .padding(.horizontal, 20)
            .padding(.top, 20)
            .padding(.bottom, 32)
        }
        .scrollIndicators(.hidden)
    }

    private func column(_ images: [GalleryImage], allImages: [GalleryImage], theme: GalleryDisplayTheme) -> some View {
        VStack(spacing: 16) {
            ForEach(images, id: \.clientId) { image in
                cell(image, allImages: allImages, theme: theme)
            }
        }
        .frame(maxWidth: .infinity)
    }

    private func cell(_ image: GalleryImage, allImages: [GalleryImage], theme: GalleryDisplayTheme) -> some View {
        let ratio = image.height > 0 ? CGFloat(image.width) / CGFloat(image.height) : 1
        // A fixed-ratio surface box (width = column width, height = width/ratio) with
        // the image filling and clipped — the reliable masonry-cell idiom.
        return RoundedRectangle(cornerRadius: 20)
            .fill(RunaColors.surface)
            .aspectRatio(min(max(ratio, 0.6), 1.6), contentMode: .fit)
            .overlay {
                GalleryImageView(image: image, theme: theme, contentMode: .fill)
            }
            .clipShape(RoundedRectangle(cornerRadius: 20))
            .contentShape(Rectangle())
            .onTapGesture {
                if let idx = allImages.firstIndex(where: { $0.clientId == image.clientId }) {
                    lightbox = LightboxContext(images: allImages, startIndex: idx, displayTheme: theme)
                }
            }
    }

    // MARK: empty / banner

    private func emptyState() -> some View {
        VStack(spacing: 28) {
            Spacer()
            NewMoonEmblem()
            Text("まだ、ひかりの記録はありません。")
                .font(RunaFonts.heading(16))
                .foregroundStyle(RunaColors.subtle)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    @ViewBuilder private func bannerLabel(_ banner: GalleryBanner) -> some View {
        if let text = bannerText(banner) {
            Text(text)
                .font(RunaFonts.body(13))
                .foregroundStyle(RunaColors.subtle)
                .frame(maxWidth: .infinity)
                .padding(.top, 12)
        }
    }

    private func bannerText(_ banner: GalleryBanner) -> String? {
        switch banner {
        case .offline: return "オフライン。取得済みの記録は、端末に残っています。"
        case .error: return "同期に、少しつまずいています。"
        default: return nil
        }
    }

    // MARK: helpers

    private func isMonotone(_ theme: GalleryDisplayTheme) -> Bool {
        switch theme {
        case .monotone: return true
        default: return false
        }
    }

    /// Split images into two columns, balancing by cumulative (clamped) height.
    private func masonry(_ images: [GalleryImage]) -> ([GalleryImage], [GalleryImage]) {
        var left: [GalleryImage] = []
        var right: [GalleryImage] = []
        var leftH: CGFloat = 0
        var rightH: CGFloat = 0
        for image in images {
            let ratio = image.height > 0 ? CGFloat(image.width) / CGFloat(image.height) : 1
            let unitHeight = 1 / min(max(ratio, 0.6), 1.6) // relative height for a unit-width cell
            if leftH <= rightH {
                left.append(image); leftH += unitHeight
            } else {
                right.append(image); rightH += unitHeight
            }
        }
        return (left, right)
    }

    private func loadPicked(_ item: PhotosPickerItem?) {
        guard let item else { return }
        Task {
            if let data = try? await item.loadTransferable(type: Data.self),
               let normalized = GalleryImageNormalizer.normalize(data) {
                model.addImage(base64: normalized.base64, width: normalized.width, height: normalized.height, mimeType: "image/jpeg")
            }
            await MainActor.run { pickerItem = nil }
        }
    }
}

/// The image itself: the presigned GET URL for an uploaded image (cached by
/// URLCache for offline viewing), or a placeholder + progress while it uploads.
struct GalleryImageView: View {
    let image: GalleryImage
    let theme: GalleryDisplayTheme
    let contentMode: ContentMode

    var body: some View {
        if let urlString = image.viewUrl, let url = URL(string: urlString) {
            AsyncImage(url: url) { img in
                img.resizable().aspectRatio(contentMode: contentMode).galleryTheme(theme)
            } placeholder: {
                RunaColors.surface
            }
        } else {
            ZStack {
                RunaColors.surface
                if !isUploaded(image.uploadState) {
                    ProgressView(value: Double(image.progress))
                        .tint(RunaColors.accent)
                        .padding(24)
                }
            }
        }
    }

    private func isUploaded(_ state: UploadState) -> Bool {
        switch state {
        case .uploaded: return true
        default: return false
        }
    }
}

/// 14 画像詳細 — the lightbox. Full-screen, swipe between images, with a ✕ close and a
/// delete affordance. Presented over a static snapshot of the grid list; deleting
/// dismisses (matching Android).
private struct LightboxContext: Identifiable {
    let id = UUID()
    let images: [GalleryImage]
    let startIndex: Int
    let displayTheme: GalleryDisplayTheme
}

private struct LightboxView: View {
    let context: LightboxContext
    let onDelete: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var selection: Int

    init(context: LightboxContext, onDelete: @escaping (String) -> Void) {
        self.context = context
        self.onDelete = onDelete
        _selection = State(initialValue: context.startIndex)
    }

    var body: some View {
        ZStack(alignment: .top) {
            RunaColors.background.ignoresSafeArea()

            TabView(selection: $selection) {
                ForEach(Array(context.images.enumerated()), id: \.element.clientId) { index, image in
                    VStack {
                        Spacer()
                        GalleryImageView(image: image, theme: context.displayTheme, contentMode: .fit)
                            .clipShape(RoundedRectangle(cornerRadius: 24))
                            .padding(.horizontal, 24)
                        Spacer().frame(height: 28)
                        Text(formatDateTime(image.createdAtEpochMs))
                            .font(RunaFonts.body(14))
                            .tracking(2)
                            .foregroundStyle(RunaColors.subtle)
                        Spacer()
                    }
                    .tag(index)
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .never))

            HStack {
                Button { dismiss() } label: {
                    Text("✕").font(RunaFonts.body(22)).foregroundStyle(RunaColors.body)
                }
                Spacer()
                Button {
                    if let image = context.images[safe: selection] {
                        onDelete(image.clientId)
                    }
                    dismiss()
                } label: {
                    Text("削除").font(RunaFonts.body(14)).foregroundStyle(RunaColors.subtle)
                }
            }
            .padding(.horizontal, 20)
            .padding(.top, 12)
        }
    }
}

/// Monotone = full desaturation; pink = a desaturate-then-tint duotone toward the
/// #F4A9C0 accent. Applied to grid cells and the lightbox alike, matching Android.
private extension View {
    @ViewBuilder func galleryTheme(_ theme: GalleryDisplayTheme) -> some View {
        switch theme {
        case .monotone:
            self.saturation(0)
        case .pink:
            self.saturation(0).colorMultiply(RunaColors.accent)
        default:
            self
        }
    }
}

private extension Array {
    subscript(safe index: Int) -> Element? {
        indices.contains(index) ? self[index] : nil
    }
}

private func formatDateTime(_ epochMs: Int64) -> String {
    let date = Date(timeIntervalSince1970: Double(epochMs) / 1000)
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "ja_JP")
    formatter.dateFormat = "M月d日  HH:mm"
    return formatter.string(from: date)
}
