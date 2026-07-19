package com.runa.android.ui.screens.gallery

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.media.ExifInterface
import android.net.Uri
import java.io.ByteArrayOutputStream

/**
 * The Android half of the gallery's OS-specific image handling: take a URI from the
 * Photo Picker and turn it into normalized JPEG bytes the shared layer can upload.
 * This is the "extract the image and hand it over" boundary — the shared
 * `GalleryViewModel.addImage` does the rest (queue → presigned PUT → register).
 *
 * Normalization: sample-decode to bound memory, apply the EXIF orientation (so a
 * portrait photo isn't stored sideways), downscale so the long edge ≤ [MAX_DIMENSION],
 * and re-encode as JPEG (keeps uploads well under the server's size cap).
 */
object ImageNormalizer {
    private const val MAX_DIMENSION = 2048
    private const val JPEG_QUALITY = 90

    data class Picked(val bytes: ByteArray, val width: Int, val height: Int, val mimeType: String)

    fun normalize(context: Context, uri: Uri): Picked? {
        val resolver = context.contentResolver

        // 1. Read bounds only, to pick a sample size.
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, bounds) } ?: return null
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null

        // 2. Decode at a memory-bounded sample size.
        val decodeOpts = BitmapFactory.Options().apply {
            inSampleSize = sampleSizeFor(bounds.outWidth, bounds.outHeight, MAX_DIMENSION * 2)
        }
        var bitmap = resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, decodeOpts) }
            ?: return null

        // 3. Correct orientation from EXIF.
        val orientation = resolver.openInputStream(uri)?.use {
            ExifInterface(it).getAttributeInt(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL)
        } ?: ExifInterface.ORIENTATION_NORMAL
        bitmap = applyOrientation(bitmap, orientation)

        // 4. Downscale so the long edge ≤ MAX_DIMENSION.
        bitmap = scaleDown(bitmap, MAX_DIMENSION)

        // 5. Re-encode as JPEG.
        val out = ByteArrayOutputStream()
        bitmap.compress(Bitmap.CompressFormat.JPEG, JPEG_QUALITY, out)
        return Picked(out.toByteArray(), bitmap.width, bitmap.height, "image/jpeg")
    }

    private fun sampleSizeFor(width: Int, height: Int, maxEdge: Int): Int {
        var sample = 1
        while (width / sample > maxEdge || height / sample > maxEdge) sample *= 2
        return sample
    }

    private fun applyOrientation(bitmap: Bitmap, orientation: Int): Bitmap {
        val matrix = Matrix()
        when (orientation) {
            ExifInterface.ORIENTATION_ROTATE_90 -> matrix.postRotate(90f)
            ExifInterface.ORIENTATION_ROTATE_180 -> matrix.postRotate(180f)
            ExifInterface.ORIENTATION_ROTATE_270 -> matrix.postRotate(270f)
            ExifInterface.ORIENTATION_FLIP_HORIZONTAL -> matrix.postScale(-1f, 1f)
            ExifInterface.ORIENTATION_FLIP_VERTICAL -> matrix.postScale(1f, -1f)
            else -> return bitmap
        }
        return Bitmap.createBitmap(bitmap, 0, 0, bitmap.width, bitmap.height, matrix, true)
    }

    private fun scaleDown(bitmap: Bitmap, maxEdge: Int): Bitmap {
        val longEdge = maxOf(bitmap.width, bitmap.height)
        if (longEdge <= maxEdge) return bitmap
        val scale = maxEdge.toFloat() / longEdge
        val w = (bitmap.width * scale).toInt().coerceAtLeast(1)
        val h = (bitmap.height * scale).toInt().coerceAtLeast(1)
        return Bitmap.createScaledBitmap(bitmap, w, h, true)
    }
}
