import UIKit

/// Turns PhotosPicker `Data` into upright, downscaled (long edge ≤ `maxDimension`) JPEG base64
/// for the shared upload.
enum GalleryImageNormalizer {
    static func normalize(_ data: Data, maxDimension: CGFloat = 2048, quality: CGFloat = 0.9) -> (base64: String, width: Int32, height: Int32)? {
        guard let image = UIImage(data: data) else { return nil }
        let scaled = image.downscaledUpright(maxDimension: maxDimension)
        guard let jpeg = scaled.jpegData(compressionQuality: quality) else { return nil }
        return (
            jpeg.base64EncodedString(),
            Int32(scaled.size.width.rounded()),
            Int32(scaled.size.height.rounded())
        )
    }
}

private extension UIImage {
    func downscaledUpright(maxDimension: CGFloat) -> UIImage {
        let longEdge = max(size.width, size.height)
        let factor = longEdge > maxDimension ? maxDimension / longEdge : 1
        let target = CGSize(width: (size.width * factor).rounded(), height: (size.height * factor).rounded())

        let format = UIGraphicsImageRendererFormat.default()
        format.scale = 1 // target is in pixels
        format.opaque = true
        return UIGraphicsImageRenderer(size: target, format: format).image { _ in
            draw(in: CGRect(origin: .zero, size: target)) // applies EXIF orientation
        }
    }
}
