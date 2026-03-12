#!/bin/bash
# ============================================================
# MuPDF-WASM Build Script
# This clones a subset of MuPDF and builds it for WASM
# ============================================================

set -e

# 1. Clone MuPDF source (using shallow clone to save space)
if [ ! -d "mupdf-src" ]; then
    echo "📥 Cloning MuPDF (shallow clone)..."
    git clone --depth 1 https://github.com/ArtifexSoftware/mupdf.git mupdf-src
    cd mupdf-src
    git submodule update --init --recursive
    cd ..
fi

# 2. Build for WASM using Emscripten
echo "🔧 Building MuPDF for WASM..."
cd mupdf-src
make OS=wasm generate
make OS=wasm libs

# 3. Export the static library
cp build/wasm/libmupdf.a ../test/
cp build/wasm/libmupdf-third.a ../test/
cd ..

# 4. Compile the final WASM module
echo "🔨 Compiling mupdf_split.wasm..."
emcc mupdf_split.c \
  -I./mupdf-src/include \
  ./test/libmupdf.a ./test/libmupdf-third.a \
  -O3 \
  -s WASM=1 \
  -s STANDALONE_WASM \
  -s EXPORTED_FUNCTIONS="['_malloc', '_free', '_invoke']" \
  --no-entry \
  -o test/mupdf_split.wasm \
  -mno-bulk-memory -mno-sign-ext -mno-nontrapping-fptoint

echo "✅ Success! Built test/mupdf_split.wasm"
