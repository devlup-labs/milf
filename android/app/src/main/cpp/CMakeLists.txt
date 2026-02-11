project(consumeronlywamr)

cmake_minimum_required(VERSION 3.10.2)

set(WAMR_ROOT_DIR ${CMAKE_CURRENT_SOURCE_DIR}/wamr)

# Set WAMR configuration
set(WAMR_BUILD_INTERP 1)
set(WAMR_BUILD_AOT 0)
set(WAMR_BUILD_JIT 0)
set(WAMR_BUILD_FAST_JIT 0)
set(WAMR_BUILD_LIBC_BUILTIN 1)
set(WAMR_BUILD_MULTI_MODULE 0)
set(WAMR_BUILD_LIB_PTHREAD 1)
set(WAMR_BUILD_LIB_WASI_THREADS 0)
set(WAMR_DISABLE_HW_BOUND_CHECK 1)

# Include WAMR VM core
include_directories(${WAMR_ROOT_DIR}/core/iwasm/include)
include_directories(${WAMR_ROOT_DIR}/core/iwasm/common)
include_directories(${WAMR_ROOT_DIR}/core/iwasm/interpreter)
include_directories(${WAMR_ROOT_DIR}/core/shared/platform/include)
include_directories(${WAMR_ROOT_DIR}/core/shared/platform/android)
include_directories(${WAMR_ROOT_DIR}/core/shared/utils)
include_directories(${WAMR_ROOT_DIR}/core/shared/mem-alloc)
include_directories(${WAMR_ROOT_DIR}/core/iwasm/libraries/libc-builtin)
include_directories(${WAMR_ROOT_DIR}/core/shared/platform/common/libc-util)

# Helper function to add source files
file(GLOB COMMON_SRC
    "${WAMR_ROOT_DIR}/core/iwasm/common/*.c"
    "${WAMR_ROOT_DIR}/core/iwasm/common/arch/*.c"
    "${WAMR_ROOT_DIR}/core/shared/platform/common/libc-util/libc_errno.c"
)
file(GLOB INTERP_SRC
    "${WAMR_ROOT_DIR}/core/iwasm/interpreter/*.c"
)
list(FILTER INTERP_SRC EXCLUDE REGEX "wasm_interp_classic.c")
list(FILTER INTERP_SRC EXCLUDE REGEX "wasm_mini_loader.c")

file(GLOB OTHER_SRC
    "${WAMR_ROOT_DIR}/core/iwasm/libraries/libc-builtin/*.c"
    "${WAMR_ROOT_DIR}/core/shared/platform/android/*.c"
    "${WAMR_ROOT_DIR}/core/shared/platform/common/posix/*.c"
    "${WAMR_ROOT_DIR}/core/shared/utils/*.c"
    "${WAMR_ROOT_DIR}/core/shared/mem-alloc/*.c"
    "${WAMR_ROOT_DIR}/core/shared/mem-alloc/ems/*.c"
)

# Add WASI support (Networking disabled)
include_directories(${WAMR_ROOT_DIR}/core/iwasm/libraries/libc-wasi)
include_directories(${WAMR_ROOT_DIR}/core/iwasm/libraries/libc-wasi/sandboxed-system-primitives/src)
include_directories(${WAMR_ROOT_DIR}/core/iwasm/libraries/libc-wasi/sandboxed-system-primitives/include)
# include_directories(${WAMR_ROOT_DIR}/core/iwasm/libraries/lib-socket/inc)

file(GLOB WASI_SRC
    "${WAMR_ROOT_DIR}/core/iwasm/libraries/libc-wasi/*.c"
    "${WAMR_ROOT_DIR}/core/iwasm/libraries/libc-wasi/sandboxed-system-primitives/src/*.c"
)

set(WAMR_CORE_SRC ${COMMON_SRC} ${INTERP_SRC} ${OTHER_SRC} ${WASI_SRC})

# Filter out unrelated platform files if GLOB picks up too much (optional, but safe for now)

add_library(
        native-lib
        SHARED
        native-lib.cpp
        memory_tracker.cpp  # Memory tracking for 512MB heap
        execution_monitor.cpp  # Execution metrics and timeout protection
        ${WAMR_CORE_SRC}
)

find_library(
        log-lib
        log
)

target_link_libraries(
        native-lib
        ${log-lib}
        m # Math library
        dl # Dynamic linking library
)

# Global definitions for all files
add_definitions(
    -DWASM_ENABLE_INTERP=1
    -DWASM_ENABLE_AOT=0
    -DWASM_ENABLE_JIT=0
    -DWASM_ENABLE_LIBC_BUILTIN=1
    -DWASM_ENABLE_FAST_JIT=0
    -DWASM_ENABLE_FAST_INTERP=1
    -DWASM_ENABLE_LIBC_WASI=1
    -DWASM_ENABLE_MODULE_INST_CONTEXT=1
    -DBH_PLATFORM_ANDROID=1
    -DBH_MALLOC=wasm_runtime_malloc
    -DBH_FREE=wasm_runtime_free
    -DWASM_HAVE_MREMAP=1
)

target_compile_definitions(native-lib PRIVATE
    # Any lib-specific flags here if needed
)
