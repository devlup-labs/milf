
#include <stdint.h>

/**
 * OBJECTIVE: CPU-Intensive Prime Number Calculation
 *
 * This program demonstrates a valid C module for WebAssembly that performs
 * a meaningful computational task. It calculates the Nth prime number.
 *
 * Why this is a "good objective":
 * 1. CPU Bound: It tests the execution speed of the runtime.
 * 2. Deterministic: The output is easily verifiable (100th prime is always
 * 541).
 * 3. No External Dependencies: It relies only on basic integer arithmetic.
 */

// Internal helper function (not exported)
static int is_prime(int n) {
  if (n <= 1)
    return 0;
  if (n <= 3)
    return 1;
  if (n % 2 == 0 || n % 3 == 0)
    return 0;
  for (int i = 5; i * i <= n; i += 6) {
    if (n % i == 0 || n % (i + 2) == 0)
      return 0;
  }
  return 1;
}

// EXPORTED FUNCTION
// This is the entry point we will call from the runtime.
// Attributes ensure it's visible to the host environment.
__attribute__((visibility("default"))) __attribute__((used)) int
find_nth_prime(int n) {
  if (n < 1)
    return -1;

  int count = 0;
  int candidate = 1;

  while (count < n) {
    candidate++;
    if (is_prime(candidate)) {
      count++;
    }
  }
  return candidate;
}

// Standard main function required by some linkers/runtimes
int main() {
  // Verify functionality by finding the 10th prime (should be 29)
  return find_nth_prime(10);
}
