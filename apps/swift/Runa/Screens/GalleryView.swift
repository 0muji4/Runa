import PhotosUI
import Shared
import SwiftUI

/// ギャラリー: two-column masonry of the user's images, rendered from the local DB.
struct GalleryView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var model = GalleryObservable()
    @State private var pickerItem: PhotosPickerItem?
    @State private var lightbox: LightboxContext?

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()
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

    // MARK: header

    private var header: some View {
        RunaScreenHeader(title: L.galleryTitle) {
            PhotosPicker(selection: $pickerItem, matching: .images) {
                Image(systemName: "plus").font(.system(size: 20)).foregroundStyle(runaTheme.subtle)
            }
        }
        .padding(.horizontal, 20)
    }

    @ViewBuilder private var content: some View {
        switch model.ui {
        case .content(let images, let sync):
            RunaSyncBanner(phase: sync)
            grid(images: images)
        case .empty:
            RunaEmptyView(
                title: L.galleryEmptyLine,
                message: L.galleryEmptyBody
            )
        case .loading:
            RunaLoadingView()
        case .failure(let error):
            RunaFailureView(error: error, onRetry: { model.refresh() })
        }
    }

    // MARK: grid (two-column masonry)

    private func grid(images: [GalleryImage]) -> some View {
        let split = masonry(images)
        return ScrollView {
            HStack(alignment: .top, spacing: 16) {
                column(split.0, allImages: images)
                column(split.1, allImages: images)
            }
            .padding(.horizontal, 20)
            .padding(.top, 20)
            .padding(.bottom, 32)
        }
        .scrollIndicators(.hidden)
    }

    private func column(_ images: [GalleryImage], allImages: [GalleryImage]) -> some View {
        VStack(spacing: 16) {
            ForEach(images, id: \.clientId) { image in
                cell(image, allImages: allImages)
            }
        }
        .frame(maxWidth: .infinity)
    }

    private func cell(_ image: GalleryImage, allImages: [GalleryImage]) -> some View {
        let ratio = image.height > 0 ? CGFloat(image.width) / CGFloat(image.height) : 1
        // Fixed-ratio box with the image filling and clipped — the masonry-cell idiom.
        return RoundedRectangle(cornerRadius: 20)
            .fill(runaTheme.surface)
            .aspectRatio(min(max(ratio, 0.6), 1.6), contentMode: .fit)
            .overlay {
                GalleryImageView(image: image, contentMode: .fill)
            }
            .clipShape(RoundedRectangle(cornerRadius: 20))
            .contentShape(Rectangle())
            .onTapGesture {
                if let idx = allImages.firstIndex(where: { $0.clientId == image.clientId }) {
                    lightbox = LightboxContext(images: allImages, startIndex: idx)
                }
            }
    }

    // MARK: helpers

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

/// The image (presigned GET URL, cached by URLCache) or a placeholder + progress while uploading.
struct GalleryImageView: View {
    @Environment(\.runaTheme) private var runaTheme
    let image: GalleryImage
    let contentMode: ContentMode

    var body: some View {
        if let urlString = image.viewUrl, let url = URL(string: urlString) {
            AsyncImage(url: url) { img in
                img.resizable().aspectRatio(contentMode: contentMode)
            } placeholder: {
                runaTheme.surface
            }
        } else {
            ZStack {
                runaTheme.surface
                if !isUploaded(image.uploadState) {
                    ProgressView(value: Double(image.progress))
                        .tint(runaTheme.accent)
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

/// Static snapshot of the grid handed to the lightbox (full-screen paging; deleting dismisses).
private struct LightboxContext: Identifiable {
    let id = UUID()
    let images: [GalleryImage]
    let startIndex: Int
}

private struct LightboxView: View {
    @Environment(\.runaTheme) private var runaTheme
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
            runaTheme.background.ignoresSafeArea()

            TabView(selection: $selection) {
                ForEach(Array(context.images.enumerated()), id: \.element.clientId) { index, image in
                    VStack {
                        Spacer()
                        GalleryImageView(image: image, contentMode: .fit)
                            .clipShape(RoundedRectangle(cornerRadius: 24))
                            .padding(.horizontal, 24)
                        Spacer().frame(height: 28)
                        Text(formatDateTime(image.createdAtEpochMs))
                            .font(RunaFonts.body(14))
                            .tracking(2)
                            .foregroundStyle(runaTheme.subtle)
                        Spacer()
                    }
                    .tag(index)
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .never))

            HStack {
                Button { dismiss() } label: {
                    Image(systemName: "xmark").font(.system(size: 18)).foregroundStyle(runaTheme.body)
                }
                Spacer()
                Button {
                    if let image = context.images[safe: selection] {
                        onDelete(image.clientId)
                    }
                    dismiss()
                } label: {
                    Text(L.galleryDelete).font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
                }
            }
            .padding(.horizontal, 20)
            .padding(.top, 12)
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
