#include "execution_monitor.h"
#include "memory_tracker.h"
#include <android/log.h>
#include <chrono>

#define LOG_TAG "ExecutionMonitor"
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, LOG_TAG, __VA_ARGS__)
#define LOGW(...) __android_log_print(ANDROID_LOG_WARN, LOG_TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)

// Static member initialization
std::chrono::steady_clock::time_point ExecutionMonitor::start_time_;
size_t ExecutionMonitor::initial_rss_bytes_ = 0;
size_t ExecutionMonitor::peak_rss_bytes_ = 0;
const char *ExecutionMonitor::current_function_ = nullptr;
uint32_t ExecutionMonitor::current_module_size_ = 0;
size_t ExecutionMonitor::current_heap_ = 0;
size_t ExecutionMonitor::current_stack_ = 0;
bool ExecutionMonitor::is_executing_ = false;

void ExecutionMonitor::StartExecution(const char *function_name,
                                      uint32_t module_size, size_t heap_size,
                                      size_t stack_size) {

  start_time_ = std::chrono::steady_clock::now();
  initial_rss_bytes_ = MemoryTracker::GetRSSBytes();
  peak_rss_bytes_ = initial_rss_bytes_;
  current_function_ = function_name;
  current_module_size_ = module_size;
  current_heap_ = heap_size;
  current_stack_ = stack_size;
  is_executing_ = true;

  LOGI("▶ Starting execution: %s (%u KB)", function_name, module_size / 1024);
  LOGI("  Heap: %zu MB, Stack: %zu MB", heap_size / (1024 * 1024),
       stack_size / (1024 * 1024));
  LOGI("  Initial RSS: %zu MB", initial_rss_bytes_ / (1024 * 1024));
}

ExecutionMonitor::Metrics ExecutionMonitor::EndExecution(int32_t result_code) {
  if (!is_executing_) {
    LOGW("EndExecution called but not executing!");
    return Metrics{};
  }

  auto end_time = std::chrono::steady_clock::now();
  auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(
                      end_time - start_time_)
                      .count();

  // Get final RSS
  size_t final_rss = MemoryTracker::GetRSSBytes();
  if (final_rss > peak_rss_bytes_) {
    peak_rss_bytes_ = final_rss;
  }

  Metrics metrics;
  metrics.start_time_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                              start_time_.time_since_epoch())
                              .count();
  metrics.end_time_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                            end_time.time_since_epoch())
                            .count();
  metrics.duration_ms = duration;
  metrics.peak_rss_bytes = peak_rss_bytes_;
  metrics.initial_rss_bytes = initial_rss_bytes_;
  metrics.heap_requested = current_heap_;
  metrics.stack_requested = current_stack_;
  metrics.module_size_bytes = current_module_size_;
  metrics.entry_function = current_function_;
  metrics.result_code = result_code;
  metrics.timeout_occurred = false;
  metrics.timeout_limit_ms =
      EstimateTimeout(current_module_size_, current_heap_);

  // Log summary
  LOGI("■ Execution Summary ─────────────────");
  LOGI("  Function: %s", current_function_);
  LOGI("  Module size: %u KB", current_module_size_ / 1024);
  LOGI("  Execution time: %llu ms", (unsigned long long)duration);
  LOGI("  Peak RSS: %zu MB", peak_rss_bytes_ / (1024 * 1024));
  LOGI("  RSS delta: %+zd MB",
       (long)((int64_t)peak_rss_bytes_ - (int64_t)initial_rss_bytes_) /
           (1024 * 1024));
  LOGI("  Heap requested: %zu MB", current_heap_ / (1024 * 1024));
  LOGI("  Stack requested: %zu MB", current_stack_ / (1024 * 1024));
  LOGI("  Result code: %d", result_code);

  // Performance warnings
  if (duration > metrics.timeout_limit_ms) {
    LOGW("⚠️  Execution time (%llu ms) exceeded recommended timeout (%u ms)",
         (unsigned long long)duration, metrics.timeout_limit_ms);
    metrics.timeout_occurred = true;
  }

  if (peak_rss_bytes_ > MemoryTracker::WARNING_THRESHOLD) {
    LOGW("⚠️  Peak memory usage high: %zu MB", peak_rss_bytes_ / (1024 * 1024));
  }

  LOGI("─────────────────────────────────────");

  is_executing_ = false;
  return metrics;
}

bool ExecutionMonitor::IsTimedOut(uint32_t timeout_ms) {
  if (!is_executing_)
    return false;

  auto current_time = std::chrono::steady_clock::now();
  auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(
                      current_time - start_time_)
                      .count();

  return duration > timeout_ms;
}

uint64_t ExecutionMonitor::GetCurrentDuration() {
  if (!is_executing_)
    return 0;

  auto current_time = std::chrono::steady_clock::now();
  return std::chrono::duration_cast<std::chrono::milliseconds>(current_time -
                                                               start_time_)
      .count();
}

uint32_t ExecutionMonitor::EstimateTimeout(uint32_t module_size,
                                           size_t heap_requested) {
  // Timeout estimation based on resources
  const uint32_t LIGHT_TIMEOUT = 5000;     // 5 seconds
  const uint32_t MEDIUM_TIMEOUT = 30000;   // 30 seconds
  const uint32_t HEAVY_TIMEOUT = 120000;   // 2 minutes
  const uint32_t EXTREME_TIMEOUT = 300000; // 5 minutes

  // Heavy workload indicators
  bool is_heavy_memory = heap_requested > (200 * 1024 * 1024);   // > 200MB
  bool is_large_module = module_size > (1024 * 1024);            // > 1MB
  bool is_extreme_memory = heap_requested > (400 * 1024 * 1024); // > 400MB

  if (is_extreme_memory) {
    LOGI("Estimated workload: EXTREME (timeout: %u ms)", EXTREME_TIMEOUT);
    return EXTREME_TIMEOUT;
  } else if (is_heavy_memory || is_large_module) {
    LOGI("Estimated workload: HEAVY (timeout: %u ms)", HEAVY_TIMEOUT);
    return HEAVY_TIMEOUT;
  } else if (heap_requested > (50 * 1024 * 1024)) { // > 50MB
    LOGI("Estimated workload: MEDIUM (timeout: %u ms)", MEDIUM_TIMEOUT);
    return MEDIUM_TIMEOUT;
  } else {
    LOGI("Estimated workload: LIGHT (timeout: %u ms)", LIGHT_TIMEOUT);
    return LIGHT_TIMEOUT;
  }
}

void ExecutionMonitor::Reset() {
  is_executing_ = false;
  current_function_ = nullptr;
  current_module_size_ = 0;
  current_heap_ = 0;
  current_stack_ = 0;
  initial_rss_bytes_ = 0;
  peak_rss_bytes_ = 0;
}
