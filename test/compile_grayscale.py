#!/usr/bin/env python3
"""
Compile grayscale.c to WASM

This script compiles the grayscale image converter to a single WASM binary.
NO external dependencies needed!
"""

import subprocess
import os
import sys

def compile_grayscale():
    """Compile grayscale.c to grayscale.wasm"""
    
    # Check if wasi-sdk is available
    wasi_clang = None
    possible_paths = [
        "/opt/wasi-sdk/bin/clang",
        "/usr/local/wasi-sdk/bin/clang",
        os.path.expanduser("~/wasi-sdk/bin/clang"),
    ]
    
    for path in possible_paths:
        if os.path.exists(path):
            wasi_clang = path
            break
    
    if not wasi_clang:
        print("❌ WASI SDK not found!")
        print("\nTo install:")
        print("1. Download: https://github.com/WebAssembly/wasi-sdk/releases")
        print("2. Extract to /opt/wasi-sdk or ~/wasi-sdk")
        print("\nOr use Docker:")
        print("  docker run --rm -v $(pwd):/src ghcr.io/webassembly/wasi-sdk:latest \\")
        print("    clang -O2 -o /src/grayscale.wasm /src/grayscale.c")
        return False
    
    print(f"✅ Found WASI SDK: {wasi_clang}")
    
    # Compile command
    cmd = [
        wasi_clang,
        "--target=wasm32-wasi",
        "-O2",                    # Optimization level 2
        "-o", "grayscale.wasm",   # Output file
        "grayscale.c"             # Input file
    ]
    
    print(f"\n🔧 Compiling: {' '.join(cmd)}")
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True)
        
        if result.returncode == 0:
            # Check file size
            size = os.path.getsize("grayscale.wasm")
            print(f"✅ Compilation successful!")
            print(f"   Output: grayscale.wasm ({size:,} bytes)")
            
            # Verify it's a valid WASM file
            with open("grayscale.wasm", "rb") as f:
                magic = f.read(4)
                if magic == b'\x00asm':
                    print("✅ Valid WASM binary (magic number correct)")
                else:
                    print("⚠️  Warning: Invalid WASM magic number")
            
            return True
        else:
            print(f"❌ Compilation failed!")
            print(f"Error: {result.stderr}")
            return False
            
    except Exception as e:
        print(f"❌ Error: {e}")
        return False

if __name__ == "__main__":
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    
    print("=" * 60)
    print("GRAYSCALE WASM COMPILER")
    print("=" * 60)
    
    success = compile_grayscale()
    
    if success:
        print("\n" + "=" * 60)
        print("✅ Ready to test!")
        print("=" * 60)
        print("\nNext steps:")
        print("1. adb push grayscale.wasm /sdcard/Download/")
        print("2. Run app and select grayscale.wasm")
        print("3. Check logs: adb logcat | grep native-lib")
    else:
        print("\n❌ Compilation failed")
        sys.exit(1)
