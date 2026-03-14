#include <stdint.h>

// Export the function so it's visible to the runtime
__attribute__((visibility("default"))) __attribute__((used)) int modulo(int a,
                                                                        int b) {
  if (b == 0)
    return 0; // Avoid division by zero panic
  return a % b;
}
