
#include <stdint.h>

// Export the function so it's visible to the runtime
__attribute__((visibility("default"))) __attribute__((used)) int add(int a,
                                                                     int b) {
  return a + b;
}

// Wrapper for generic runner (takes 0 args)
__attribute__((visibility("default"))) __attribute__((used)) int app_main() {
  return add(50, 50);
}
