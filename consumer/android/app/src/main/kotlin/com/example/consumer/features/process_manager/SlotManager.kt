package com.example.consumer.features.process_manager

import com.example.consumer.features.wasm_runtime.WasmEngine
import com.example.consumer.features.wasm_runtime.MemoryLimitExceededException
import com.example.consumer.features.os_stats.OsReader
import kotlinx.coroutines.*
import java.io.File

class SlotManager(
    private val wasmEngine: WasmEngine = WasmEngine(),
    private val osReader: OsReader,
    private val filesDir: File,
    private val ioScope: CoroutineScope = CoroutineScope(Dispatchers.Default + SupervisorJob())
) {

    fun checkAvailableProcesses(): List<String> {
        return emptyList()
    }

    fun startProcessSlot(
        wasmBinary: ByteArray, 
        input: Map<String, Any>, 
        metadata: Map<String, Any>,
        memoryLimitMB: Int = 300, // Default 300MB
        onComplete: (String, String, Long) -> Unit // path, content, memoryDelta
    ) {
        ioScope.launch {
            try {
                withTimeout(10000L) { // Increased timeout to accommodate simulated growth
                    val inputBytes = input.toString().toByteArray() 
                    
                    // Measure before
                    val memBefore = osReader.getAppMemory()
                    
                    val resultString = wasmEngine.execute(wasmBinary, inputBytes, memoryLimitMB)
                    
                    // Measure after
                    val memAfter = osReader.getAppMemory()
                    val memDelta = if (memAfter > memBefore) memAfter - memBefore else 0L
                    
                    val outputFile = File(filesDir, "wasm_output_${System.currentTimeMillis()}.txt")
                    outputFile.writeText(resultString)
                    
                    onComplete(outputFile.absolutePath, resultString, memDelta)
                }
            } catch (e: MemoryLimitExceededException) {
                onComplete("", "Memory Error: ${e.message}", 0L)
            } catch (e: Throwable) {
                onComplete("", "Error: ${e.message ?: e.toString()}", 0L)
            }
        }
    }

    fun stopProcessSlot(pid: String) {}
}
