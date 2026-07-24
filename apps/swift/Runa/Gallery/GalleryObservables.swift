import Foundation
import Shared

/// ObservableObject bridge over the shared `GalleryViewModel` (13 ギャラリー). Mirrors
/// the other observables: collect the SKIE-bridged `StateFlow` and republish on the
/// main actor; the display-theme toggle and mutations forward to the shared VM.
/// The gallery grid's page state, mapped from the shared `UiState<List<GalleryImage>>`
/// into a native Swift enum so the view never touches SKIE generics. The display-theme
/// toggle is a separate flow (grid chrome shown over both content and empty).
enum GalleryUi {
    case loading
    case empty
    case content(images: [GalleryImage], sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `GalleryViewModel` (13 ギャラリー). Collects
/// the SKIE-bridged `StateFlow<UiState<…>>` + the display-theme flow, mapping each grid
/// emission to [GalleryUi], and republishes on the main actor; mutations forward to the
/// shared VM.
@MainActor
final class GalleryObservable: ObservableObject {
    @Published private(set) var ui: GalleryUi = .loading
    @Published private(set) var displayTheme: GalleryDisplayTheme = .pink

    private let viewModel: GalleryViewModel
    private var collectTask: Task<Void, Never>?
    private var themeTask: Task<Void, Never>?

    init(viewModel: GalleryViewModel = resolveGalleryViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.viewModel.state {
                switch runaDecode(value, as: [GalleryImage].self) {
                case .loading: self.ui = .loading
                case .empty: self.ui = .empty
                case .content(let images, let sync): self.ui = .content(images: images, sync: sync)
                case .failure(let error): self.ui = .failure(error)
                }
            }
        }
        themeTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.viewModel.displayTheme {
                self.displayTheme = value
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

    deinit {
        collectTask?.cancel()
        themeTask?.cancel()
    }
}
