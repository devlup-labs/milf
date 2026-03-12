#ifndef EXECUTION_MONITOR_H
#define EXECUTION_MONITOR_H

#include <chrono>
#include <cstddef>
#include <cstdint>

/**
 * Execution Monitor - Tracks WASM execution metrics
 *
 * Monitors:
 * - Execution time (for timeout detection)
 * - Memory usage (peak RSS)
 * - Function being executed
 * - Result codes
 *
 * Supports heavy workloads with timeout protection
 */
class ExecutionMonitor {
public:
  struct Metrics {
    // Timing
    uint64_t start_time_ms;
    uint64_t end_time_ms;
    uint64_t duration_ms;

    // Memory
    size_t peak_rss_bytes;
    size_t initial_rss_bytes;
    size_t heap_requested;
    size_t stack_requested;

    // Module info
    uint32_t module_size_bytes;
    const char *entry_function;
    int32_t result_code;

    // Timeout tracking
    bool timeout_occurred;
    uint32_t timeout_limit_ms;
  };

  /**
   * Start tracking execution
   * @param function_name Entry point being called (e.g., "app_main")
   * @param module_size Size of WASM module in bytes
   * @param heap_size Heap allocated for this execution
   * @param stack_size Stack allocated for this execution
   */
  static void StartExecution(const char *function_name, uint32_t module_size,
                             size_t heap_size, size_t stack_size);

  /**
   * End tracking and return metrics
   * @param result_code Return code from WASM execution
   * @return Complete metrics for this execution
   */
  static Metrics EndExecution(int32_t result_code);

  /**
   * Check if current execution has exceeded timeout
   * @param timeout_ms Timeout in milliseconds
   * @return true if timed out
   */
  static bool IsTimedOut(uint32_t timeout_ms);

  /**
   * Get current execution duration
   * @return Milliseconds since StartExecution
   */
  static uint64_t GetCurrentDuration();

  /**
   * Estimate appropriate timeout based on resource requirements
   * @param module_size Size of WASM module
   * @param heap_requested Heap memory requested
   * @return Recommended timeout in milliseconds
   */
  static uint32_t EstimateTimeout(uint32_t module_size, size_t heap_requested);

  /**
   * Reset monitor state
   */
  static void Reset();

private:
  static std::chrono::steady_clock::time_point start_time_;
  static size_t initial_rss_bytes_;
  static size_t peak_rss_bytes_;
  static const char *current_function_;
  static uint32_t current_module_size_;
  static size_t current_heap_;
  static size_t current_stack_;
  static bool is_executing_;
};

#endif // EXECUTION_MONITOR_H
