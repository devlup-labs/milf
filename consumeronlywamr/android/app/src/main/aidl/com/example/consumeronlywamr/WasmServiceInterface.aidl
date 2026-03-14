// WasmServiceInterface.aidl
package com.example.consumeronlywamr;

// Declare any non-default types here with import statements

interface WasmServiceInterface {
    /**
     * Executes WASM bytecode in the isolated process.
     * Returns the output string or error message.
     */
    String runWasm(in byte[] wasmBytes);
    
    /**
     * Executes a specific WASM function by name with a list of integer arguments.
     * This is the "Generic Dispatcher" that allows running any WASM module.
     */
    int invokeWasm(in byte[] wasmBytes, String funcName, in int[] args);

    // Old hardcoded method for reference/compatibility
    int wasmAdd(in byte[] wasmBytes, int a, int b);

    /**
     * Universal Dispatcher for input using ABI Pattern:
     * Passes a byte payload into WASM linear memory and expects a byte array back.
     */
    byte[] invokeDataWasm(in byte[] wasmBytes, String funcName, in byte[] payload, String hostPath);
}
