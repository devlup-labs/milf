import struct

def create_modulo_wasm():
    # WASM Binary Format
    # ------------------
    
    # 1. Header: Magic \0asm + Version 1
    wasm_bytes = b'\x00\x61\x73\x6D\x01\x00\x00\x00'
    
    # 2. Type Section (ID 1)
    # 1 type: (i32, i32) -> i32
    # Bytes: 01 (ID) 07 (Size) 01 (Count) 60 (Func) 02 (Params) 7F (i32) 7F (i32) 01 (Results) 7F (i32)
    wasm_bytes += b'\x01\x07\x01\x60\x02\x7F\x7F\x01\x7F'
    
    # 3. Function Section (ID 3)
    # 1 function using Type Index 0
    # Bytes: 03 (ID) 02 (Size) 01 (Count) 00 (Type Index)
    wasm_bytes += b'\x03\x02\x01\x00'
    
    # 4. Export Section (ID 7)
    # Export "modulo" -> Func Index 0
    # Bytes: 07 (ID) 0A (Size) 01 (Count) 06 (NameLen) "modulo" 00 (Kind Func) 00 (Index)
    wasm_bytes += b'\x07\x0A\x01\x06\x6D\x6F\x64\x75\x6C\x6F\x00\x00'
    
    # 5. Code Section (ID 10)
    # Body for Func 0
    # Logic: local.get 0, local.get 1, i32.rem_s, end
    # Bytes: 0A (ID) 09 (Size) 01 (Count) 07 (FuncSize) 00 (Locals) 20 00 (Get 0) 20 01 (Get 1) 6F (Rem_s) 0B (End)
    wasm_bytes += b'\x0A\x09\x01\x07\x00\x20\x00\x20\x01\x6F\x0B'
    
    # Write to file
    with open('test/modulo.wasm', 'wb') as f:
        f.write(wasm_bytes)
    print("✅ Generated test/modulo.wasm (Manual Bytecode)")

if __name__ == "__main__":
    create_modulo_wasm()
