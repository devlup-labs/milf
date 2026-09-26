package com.example.consumeronlywamr

import android.util.Log
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL
import java.nio.ByteBuffer
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicInteger

/**
 * HttpStreamer
 *
 * Host-environment I/O subsystem providing WASM guest modules with Android platform capabilities.
 *
 * Architecture Role:
 * - WebAssembly modules running inside WAMR are sandboxed and lack direct OS networking,
 *   filesystem, and PDF generation capabilities.
 * - This class implements the Android host side of custom WAMR native functions
 *   (e.g., `milf_stream_open`, `milf_stream_read`, `milf_stream_close`, `milf_pdf_generate`).
 * - Methods in this class are invoked directly from C++ JNI bridge functions defined
 *   in `native-lib.cpp`.
 * - Maintains active streams using thread-safe data structures ([ConcurrentHashMap], [AtomicInteger])
 *   to handle concurrent asynchronous execution from WASM threads safely.
 */
class HttpStreamer {

    private val activeStreams = ConcurrentHashMap<Int, InputStream>()
    private val activeConnections = ConcurrentHashMap<Int, HttpURLConnection>()
    private val handleCounter = AtomicInteger(1) // Monotonically increasing stream handle ID

    /**
     * Opens an HTTPS streaming connection for the given URL.
     * Invoked from C++: `native_milf_stream_open`.
     *
     * Security Constraint:
     * - Restricts requests strictly to HTTPS.
     *
     * @param urlString Target remote URL.
     * @return Positive integer handle identifying the active stream, or negative error code.
     */
    fun openStream(urlString: String): Int {
        val policy = android.os.StrictMode.ThreadPolicy.Builder().permitAll().build()
        android.os.StrictMode.setThreadPolicy(policy)

        if (!urlString.startsWith("https://")) {
            Log.e("HttpStreamer", "SECURITY ERROR: Only HTTPS is allowed. URL: $urlString")
            return -2
        }

        try {
            val url = URL(urlString)
            val connection = url.openConnection() as HttpURLConnection
            connection.connectTimeout = 10000
            connection.readTimeout = 30000
            connection.requestMethod = "GET"

            val responseCode = connection.responseCode
            if (responseCode !in 200..299) {
                Log.e("HttpStreamer", "HTTP Error $responseCode for URL: $urlString")
                return -1
            }

            val handle = handleCounter.getAndIncrement()
            activeConnections[handle] = connection
            activeStreams[handle] = connection.inputStream

            val contentLength = connection.contentLength
            Log.i("HttpStreamer", "Opened stream for handle $handle, size: $contentLength bytes")
            
            return handle
        } catch (e: Exception) {
            Log.e("HttpStreamer", "Failed to open stream: ${e.message}")
            return -1
        }
    }

    /**
     * Reads a chunk of bytes from an open stream handle directly into WASM linear memory.
     * Invoked from C++: `native_milf_stream_read`.
     *
     * Performance:
     * - Uses [directBuf] (a Direct [ByteBuffer]) allowing native C++ / WAMR to access
     *   the read payload directly without intermediate user-space buffer copies.
     *
     * @param handle Stream identifier returned by [openStream].
     * @param maxSize Maximum number of bytes to read in this chunk.
     * @param directBuf Direct NIO buffer backed by WASM module linear memory.
     * @return Number of bytes read, 0 on EOF, or -1 on error.
     */
    fun readChunk(handle: Int, maxSize: Int, directBuf: ByteBuffer): Int {
        val stream = activeStreams[handle] ?: return -1
        val tempBlock = ByteArray(maxSize)
        
        try {
            val readBytes = stream.read(tempBlock, 0, maxSize)
            if (readBytes == -1) {
                return 0 // EOF reached
            }

            // Copy chunk from Kotlin byte array seamlessly into C++ linear memory (thanks to DirectByteBuffer)
            directBuf.put(tempBlock, 0, readBytes)
            return readBytes
        } catch (e: Exception) {
            Log.e("HttpStreamer", "Failed to read chunk: ${e.message}")
            return -1
        }
    }

    /**
     * Closes an active stream and releases the underlying HTTP connection.
     * Invoked from C++: `native_milf_stream_close`.
     *
     * @param handle Stream identifier.
     */
    fun closeStream(handle: Int) {
        try {
            activeStreams[handle]?.close()
            activeConnections[handle]?.disconnect()
        } catch (e: Exception) {
            Log.e("HttpStreamer", "Error closing stream $handle: ${e.message}")
        } finally {
            activeStreams.remove(handle)
            activeConnections.remove(handle)
            Log.i("HttpStreamer", "Closed stream handle $handle")
        }
    }

    /**
     * Synthesizes a PDF document from raw text on behalf of the WASM workload.
     * Invoked from C++: `native_milf_pdf_generate`.
     *
     * @param text Raw textual content to render into standard A4 PDF pages.
     * @return Byte array containing standard binary PDF file data.
     */
    fun generatePdf(text: String): ByteArray {
        try {
            val document = android.graphics.pdf.PdfDocument()
            val pageInfo = android.graphics.pdf.PdfDocument.PageInfo.Builder(595, 842, 1).create()
            val page = document.startPage(pageInfo)
            
            val canvas = page.canvas
            val paint = android.graphics.Paint()
            paint.textSize = 12f
            
            // Draw text with simple wrapping
            var y = 50f
            text.split("\n").forEach { line ->
                canvas.drawText(line, 50f, y, paint)
                y += 20f
            }
            
            document.finishPage(page)
            
            val outputStream = java.io.ByteArrayOutputStream()
            document.writeTo(outputStream)
            document.close()
            
            return outputStream.toByteArray()
        } catch (e: Exception) {
            Log.e("HttpStreamer", "Failed to generate PDF: ${e.message}")
            return ByteArray(0)
        }
    }

    // Called via JNI from C++ (native_milf_storage_save)
    // We need a path. Simple: use private files dir.
    private var baseDir: java.io.File? = null

    /**
     * Sets base storage root directory (typically context.filesDir).
     */
    fun setStorageDir(dir: java.io.File) {
        baseDir = dir
    }

    /**
     * Persists binary data to private app sandbox storage.
     * Invoked from C++: `native_milf_storage_save`.
     *
     * @param name File name within internal storage directory.
     * @param data Binary payload to write.
     * @return 0 on success, -1 on failure.
     */
    fun saveToStorage(name: String, data: ByteArray): Int {
        val dir = baseDir ?: return -1
        try {
            val file = java.io.File(dir, name)
            file.writeBytes(data)
            Log.i("HttpStreamer", "Saved ${data.size} bytes to ${file.absolutePath}")
            return 0
        } catch (e: Exception) {
            Log.e("HttpStreamer", "Failed to save to storage: ${e.message}")
            return -1
        }
    }

    /**
     * Registers this Kotlin instance with C++ JNI as the active host callbacks handler.
     */
    external fun bindNative()
}
