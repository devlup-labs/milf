#include <stdint.h>

__attribute__((visibility("default"))) __attribute__((used)) 
int wasm_main(int a, int b) {
  return a + b;
}
