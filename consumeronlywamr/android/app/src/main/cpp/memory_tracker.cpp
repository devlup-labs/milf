#include "memory_tracker.h"
#include <android/log.h>
#include <fstream>
#include <sstream>
#include <string>

#define LOG_TAG "MemoryTracker"
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, LOG_TAG, __VA_ARGS__)
#define LOGW(...) __android_log_print(ANDROID_LOG_WARN, LOG_TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)

// Static member initialization
std::atomic<size_t> MemoryTracker::total_allocated_{0};
std::atomic<size_t> MemoryTracker::total_freed_{0};
bool MemoryTracker::initialized_ = false;

void MemoryTracker::Initialize() {
  if (initialized_)
    return;

  ResetCounters();
  initialized_ = true;

  LOGI("Memory tracker initialized");
  LOGI("  Max heap: %zu MB", MAX_HEAP_BYTES / (1024 * 1024));
  LOGI("  Max stack: %zu MB", MAX_STACK_BYTES / (1024 * 1024));
  LOGI("  Warning threshold: %zu MB", WARNING_THRESHOLD / (1024 * 1024));
  LOGI("  Hard limit: %zu MB", MAX_TOTAL / (1024 * 1024));

  // Log current RSS
  size_t current_rss = GetRSSBytes();
  LOGI("  Current RSS: %zu MB", current_rss / (1024 * 1024));
}

void MemoryTracker::RecordAllocation(size_t bytes) {
  total_allocated_.fetch_add(bytes);
  size_t current = GetCurrentUsage();
  size_t rss = GetRSSBytes();

  LOGI("Allocated %zu MB (total tracked: %zu MB, RSS: %zu MB)",
       bytes / (1024 * 1024), current / (1024 * 1024), rss / (1024 * 1024));

  if (rss > WARNING_THRESHOLD) {
    LOGW("⚠️  Memory usage high: %zu MB (warning threshold: %zu MB)",
         rss / (1024 * 1024), WARNING_THRESHOLD / (1024 * 1024));
  }

  if (rss > MAX_TOTAL) {
    LOGE("🚨 MEMORY LIMIT EXCEEDED: %zu MB (max: %zu MB)", rss / (1024 * 1024),
         MAX_TOTAL / (1024 * 1024));
  }
}

void MemoryTracker::RecordDeallocation(size_t bytes) {
  total_freed_.fetch_add(bytes);
  size_t current = GetCurrentUsage();

  LOGI("Deallocated %zu MB (remaining: %zu MB)", bytes / (1024 * 1024),
       current / (1024 * 1024));
}

/**
 * Returns net tracked bytes allocated specifically for WASM modules.
 * Note: Lockless read using sequentially consistent atomic loads.
 */
size_t MemoryTracker::GetCurrentUsage() {
  size_t allocated = total_allocated_.load();
  size_t freed = total_freed_.load();
  return allocated > freed ? (allocated - freed) : 0;
}

/**
 * Parses Linux Kernel procfs (`/proc/self/status`) for `VmRSS` (Resident Set Size).
 *
 * Research Rationale:
 * - Unlike standard heap profiling (`mallinfo` or `sbrk`), VmRSS reflects actual physical
 *   RAM pages mapped into the process's page table by the MMU, including:
 *   1. Anonymous memory (WASM linear memory pages committed by jemalloc/scudo).
 *   2. File-backed memory (shared libraries like libnative-lib.so and Android framework).
 *   3. Shared memory mappings.
 * - This provides the exact metric inspected by Android's LMKD when determining OOM kills.
 */
size_t MemoryTracker::GetRSSBytes() {
  std::ifstream status("/proc/self/status");
  if (!status.is_open()) {
    LOGW("Failed to open /proc/self/status");
    return 0;
  }

  std::string line;
  while (std::getline(status, line)) {
    if (line.find("VmRSS:") == 0) {
      // Parse "VmRSS:   12345 kB"
      std::istringstream iss(line.substr(6)); // Skip "VmRSS:"
      size_t kb;
      if (iss >> kb) {
        return kb * 1024;
      }
    }
  }

  return 0;
}

/**
 * Evaluates whether current RSS has crossed the high-watermark threshold (80% of MAX_TOTAL = 1.2 GB).
 *
 * If true, subsequent module loads/instantiations are aborted immediately to prevent
 * an unrecoverable SIGKILL from Android's LMKD.
 */
bool MemoryTracker::IsNearLimit() {
  size_t rss = GetRSSBytes();
  size_t threshold =
      static_cast<size_t>(MAX_TOTAL * 0.8); // 80% of 1.5GB = 1.2GB

  if (rss > threshold) {
    LOGW("Near memory limit: %zu MB / %zu MB (%.1f%%)", rss / (1024 * 1024),
         MAX_TOTAL / (1024 * 1024), 100.0 * rss / MAX_TOTAL);
    return true;
  }

  return false;
}

void MemoryTracker::ResetCounters() {
  total_allocated_.store(0);
  total_freed_.store(0);
  LOGI("Memory counters reset");
}
