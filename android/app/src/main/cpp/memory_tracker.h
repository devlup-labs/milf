#ifndef MEMORY_TRACKER_H
#define MEMORY_TRACKER_H

#include <atomic>
#include <cstddef>

/**
 * Memory Tracker - Ensures WASM execution stays within device limits
 *
 * Device constraints:
 * - Average: 500MB-1GB
 * - Maximum: 1.5GB (hard limit)
 *
 * WASM allocation:
 * - Heap: 512MB (safe within budget)
 * - Stack: 16MB
 * - Total WASM: ~530MB
 * - Leaves ~500MB for system/Flutter
 */
class MemoryTracker {
public:
  // Memory limits (adjustable via config later)
  static const size_t MAX_HEAP_BYTES = 512 * 1024 * 1024;     // 512MB
  static const size_t MAX_STACK_BYTES = 16 * 1024 * 1024;     // 16MB
  static const size_t WARNING_THRESHOLD = 1024 * 1024 * 1024; // 1GB
  static const size_t MAX_TOTAL = 1536 * 1024 * 1024; // 1.5GB hard limit

  /**
   * Initialize memory tracking
   */
  static void Initialize();

  /**
   * Record WASM module allocation
   */
  static void RecordAllocation(size_t bytes);

  /**
   * Record WASM module deallocation
   */
  static void RecordDeallocation(size_t bytes);

  /**
   * Get current tracked usage
   */
  static size_t GetCurrentUsage();

  /**
   * Get actual RSS from /proc/self/status
   */
  static size_t GetRSSBytes();

  /**
   * Check if near memory limit (>80% of 1.5GB)
   */
  static bool IsNearLimit();

  /**
   * Reset tracking counters
   */
  static void ResetCounters();

  /**
   * Get heap and stack limits for WASM instantiation
   */
  static size_t GetMaxHeap() { return MAX_HEAP_BYTES; }
  static size_t GetMaxStack() { return MAX_STACK_BYTES; }

private:
  static std::atomic<size_t> total_allocated_;
  static std::atomic<size_t> total_freed_;
  static bool initialized_;
};

#endif // MEMORY_TRACKER_H
