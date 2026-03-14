
import struct

def create_wasm():
    # Magic (4) + Version (4)
    wasm_bytes = b'\x00\x61\x73\x6D\x01\x00\x00\x00'
    
    # Type Section (1): 1 type, func() -> void
    wasm_bytes += b'\x01\x04\x01\x60\x00\x00'
    
    # Function Section (3): 1 func, type index 0
    wasm_bytes += b'\x03\x02\x01\x00'
    
    # Export Section (7): 1 export, "main", func index 0
    # "main" len 4 -> \x6D\x61\x69\x6E
    wasm_bytes += b'\x07\x08\x01\x04\x6D\x61\x69\x6E\x00\x00'
    
    # Code Section (10): 1 func body
    # locals count 0, opcode end (0x0B)
    wasm_bytes += b'\x0A\x04\x01\x02\x00\x0B'
    
    with open('test/valid.wasm', 'wb') as f:
        f.write(wasm_bytes)
    print("Created test/valid.wasm")

if __name__ == "__main__":
    create_wasm()
