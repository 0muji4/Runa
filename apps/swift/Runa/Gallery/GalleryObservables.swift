import Foundation
import Shared

/// ObservableObject bridge over the shared `GalleryViewModel` (13 ギャラリー). Mirrors
/// the other observables: collect the SKIE-bridged `StateFlow` and republish on the
/// main actor; the display-theme toggle and mutations forward to the shared VM.
@MainActor
final class GalleryObservable: ObservableObject {
    @Published private(set) var state: GalleryUiState?

    private let viewModel: GalleryViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: GalleryViewModel = resolveGalleryViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<GalleryUiState> = self.viewModel.state
            for await value in flow {
                self.state = value
            }
        }
    }

    /// Switch the gallery-scoped display treatment (monotone ⇔ pink). Client-only —
    /// NOT the app-wide theme.
    func setDisplayTheme(_ theme: GalleryDisplayTheme) {
        viewModel.setDisplayTheme(theme: theme)
    }

    /// Add a picked image. `base64` is the normalized JPEG; the shared helper decodes
    /// it to a Kotlin `ByteArray` reference, so no per-byte bridging happens in Swift.
    func addImage(base64: String, width: Int32, height: Int32, mimeType: String) {
        let bytes = galleryDecodeBase64(value: base64)
        viewModel.addImage(bytes: bytes, width: width, height: height, mimeType: mimeType)
    }

    func deleteImage(clientId: String) {
        viewModel.deleteImage(clientId: clientId)
    }

    func refresh() {
        viewModel.refresh()
    }

    deinit { collectTask?.cancel() }
}
