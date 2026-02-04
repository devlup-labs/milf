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
     * Executes 'add' function in WASM with two integer inputs.
     */
    int wasmAdd(in byte[] wasmBytes, int a, int b);
}
