import UIKit

/// The iOS half of the gallery's OS-specific image handling: take the `Data` from a
/// PhotosPicker item and turn it into normalized JPEG bytes (base64) the shared layer
/// can upload. This is the "extract the image and hand it over" boundary — the shared
/// `GalleryViewModel.addImage` does the rest (queue → presigned PUT → register).
///
/// Normalization: draw the image upright (`UIImage.draw` applies the EXIF orientation),
/// downscale so the long edge ≤ `maxDimension`, and re-encode as JPEG (keeps uploads
/// well under the server's size cap).
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
