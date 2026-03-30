/**
 * milf.h — MILF Serverless Platform Standard Library
 *
 * Provides basic type definitions and utility functions for WASM functions
 * compiled with -nostdlib. Include this in any C function:
 *
 *   #include <milf.h>
 */

#ifndef MILF_H
#define MILF_H

/* ── Type Definitions ──────────────────────────────────────────────────── */

typedef signed char        int8_t;
typedef unsigned char      uint8_t;
typedef signed short       int16_t;
typedef unsigned short     uint16_t;
typedef signed int         int32_t;
typedef unsigned int       uint32_t;
typedef signed long long   int64_t;
typedef unsigned long long uint64_t;
typedef unsigned int       size_t;

#ifndef NULL
#define NULL ((void *)0)
#endif

/* ── Export Helper ─────────────────────────────────────────────────────── */

/**
 * Mark a function as exported from the WASM module.
 *
 * Usage:
 *   MILF_EXPORT int add(int a, int b) { return a + b; }
 */
#define MILF_EXPORT \
    __attribute__((visibility("default"))) \
    __attribute__((used))

/* ── String Utilities ──────────────────────────────────────────────────── */

static inline size_t milf_strlen(const char *s) {
    size_t len = 0;
    while (s[len]) len++;
    return len;
}

/* ── Memory Utilities ──────────────────────────────────────────────────── */

static inline void *milf_memcpy(void *dest, const void *src, size_t n) {
    unsigned char *d = (unsigned char *)dest;
    const unsigned char *s = (const unsigned char *)src;
    while (n--) *d++ = *s++;
    return dest;
}

static inline void *milf_memset(void *dest, int val, size_t n) {
    unsigned char *d = (unsigned char *)dest;
    while (n--) *d++ = (unsigned char)val;
    return dest;
}

static inline int milf_memcmp(const void *a, const void *b, size_t n) {
    const unsigned char *pa = (const unsigned char *)a;
    const unsigned char *pb = (const unsigned char *)b;
    while (n--) {
        if (*pa != *pb) return *pa - *pb;
        pa++;
        pb++;
    }
    return 0;
}

/* ── Math Helpers ──────────────────────────────────────────────────────── */

static inline int32_t milf_abs(int32_t x) {
    return x < 0 ? -x : x;
}

static inline int32_t milf_min(int32_t a, int32_t b) {
    return a < b ? a : b;
}

static inline int32_t milf_max(int32_t a, int32_t b) {
    return a > b ? a : b;
}

#endif /* MILF_H */
