/**
 * Simple Fibonacci - No stdlib needed
 * Tests basic CPU computation and timeout estimation
 */

// Export for WASM
__attribute__((visibility("default"))) int fibonacci(int n) {
  if (n <= 1)
    return n;
  return fibonacci(n - 1) + fibonacci(n - 2);
}

// Main entry point
__attribute__((visibility("default"))) int app_main() {
  // Calculate 35th fibonacci (takes ~100-500ms)
  // Tests MEDIUM timeout tier
  return fibonacci(35);
}
