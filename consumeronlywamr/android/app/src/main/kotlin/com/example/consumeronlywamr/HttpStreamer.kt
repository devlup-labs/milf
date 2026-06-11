package com.example.consumeronlywamr

import android.util.Log
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL
import java.nio.ByteBuffer
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicInteger

class HttpStreamer {

    private val activeStreams = ConcurrentHashMap<Int, InputStream>()
    private val activeConnections = ConcurrentHashMap<Int, HttpURLConnection>()
    private val handleCounter = AtomicInteger(1) // Always positive

    // Called via JNI from C++ (native_milf_stream_open)
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

    // Called via JNI from C++ (native_milf_stream_read)
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

    // Called via JNI from C++ (native_milf_stream_close)
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

    // Called via JNI from C++ (native_milf_pdf_generate)
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

    fun setStorageDir(dir: java.io.File) {
        baseDir = dir
    }

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

    // Connect this Kotlin instance to the C++ global JNI references
    external fun bindNative()
}
