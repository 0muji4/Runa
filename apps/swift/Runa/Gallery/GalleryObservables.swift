import Foundation
import Shared

/// The gallery grid's page state, decoded from the shared `UiState<List<GalleryImage>>`.
enum GalleryUi {
    case loading
    case empty
    case content(images: [GalleryImage], sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `GalleryViewModel`.
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

    /// Gallery-scoped display treatment (monotone ⇔ pink); NOT the app-wide theme.
    func setDisplayTheme(_ theme: GalleryDisplayTheme) {
        viewModel.setDisplayTheme(theme: theme)
    }

    /// Add a picked image; `base64` is the normalized JPEG, decoded to `ByteArray` in Kotlin.
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
