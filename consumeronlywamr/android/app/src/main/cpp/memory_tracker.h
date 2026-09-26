#ifndef MEMORY_TRACKER_H
#define MEMORY_TRACKER_H

#include <atomic>
#include <cstddef>

/**
 * MemoryTracker
 *
 * Real-time memory watchdog and quota enforcement subsystem for WASM execution.
 *
 * Architectural & OS Research Context:
 * 1. Android Low Memory Killer Daemon (LMKD):
 *    - Android kernel monitors memory pressure through memory cgroups (memcg) and psi
 *      (Pressure Stall Information). When system RAM drops below critical watermarks,
 *      LMKD terminates processes based on their `oom_score_adj`.
 *    - While our `:wasm_engine` process is isolated, exceeding ~1.5 GB RSS triggers
 *      aggressive OOM termination by LMKD.
 *
 * 2. Resident Set Size (RSS) vs Virtual Memory Size (VSS):
 *    - In WebAssembly runtimes, `memory.grow` reserves virtual address space.
 *    - VSS (Virtual Set Size) includes uncommitted pages that do not consume physical RAM.
 *    - RSS (Resident Set Size) represents physical RAM pages currently mapped into the
 *      page tables. Querying `/proc/self/status` (VmRSS) yields true physical memory impact.
 *
 * 3. WASM Linear Memory Budgeting:
 *    - MAX_HEAP_BYTES (512 MB): Upper bound for module instance heap allocation.
 *    - MAX_STACK_BYTES (16 MB): Shadow execution stack for deep recursion / call frames.
 *    - WARNING_THRESHOLD (1.0 GB): Soft watermark to throttle incoming tasks.
 *    - MAX_TOTAL (1.5 GB): Hard ceiling beyond which further module instantiations are aborted.
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
