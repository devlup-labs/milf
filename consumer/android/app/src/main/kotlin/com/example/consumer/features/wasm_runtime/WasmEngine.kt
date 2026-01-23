package com.example.consumer.features.wasm_runtime

/**
 * Responsible for the actual execution of WebAssembly binaries.
 * Only handles the execution context, not the process/slot lifecycle.
 */
class MemoryLimitExceededException(message: String) : Exception(message)

class WasmEngine {

    /**
     * Executes the given WASM binary with the provided input.
     * 
     * @param memoryLimitMB: Maximum RAM allowed for this execution
     */
    fun execute(binary: ByteArray, input: ByteArray, memoryLimitMB: Int): String {
        val inputStr = String(input)
        
        val requiredMB = when {
            inputStr.contains("heavy") -> 100
            inputStr.contains("medium") -> 50
            else -> (20..80).random() 
        }

        println("WASM Slot: Job requires $requiredMB MB. Limit is $memoryLimitMB MB.")

        // Simulate memory growth
        for (i in 1..requiredMB) {
            if (i > memoryLimitMB) {
                throw MemoryLimitExceededException("Process terminated: Memory limit of ${memoryLimitMB}MB exceeded (Attempted to use ${i}MB)")
            }
            // Simulate allocation
            if (i % 10 == 0) Thread.sleep(100) // Simulate work
        }
        
        // Final allocation
        val simulatedHeap = ByteArray(requiredMB * 1024 * 1024)
        simulatedHeap.fill(1)
        
        Thread.sleep(1000L) 
        
        val checkVal = simulatedHeap[0]
        
        return """
            [WASM RUNTIME REPORT]
            Target: $inputStr
            Status: EXECUTION_SUCCESS
            
            Resource Usage:
              - Peak Simulated Heap: ${requiredMB} MB
              - Memory Limit: ${memoryLimitMB} MB
              - Dirty Pages Check: $checkVal
            
            Telemetry:
            Timestamp: ${System.currentTimeMillis()}
        """.trimIndent()
    }
}
