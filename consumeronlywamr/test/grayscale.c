/**
 * IMAGE GRAYSCALE CONVERTER - Single Binary WASM
 *
 * This is a self-contained WASM module that converts RGB images to grayscale.
 * No external libraries needed - everything is in this one file!
 *
 * Lambda equivalent: AWS Lambda image processing function
 *
 * Compile: wasi-sdk/bin/clang --target=wasm32-wasi -O2 -o grayscale.wasm
 * grayscale.c
 */

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

// Image data structure
typedef struct {
  uint32_t width;
  uint32_t height;
  uint8_t *data; // RGB data (3 bytes per pixel)
} Image;

/**
 * Convert RGB pixel to grayscale using luminosity method
 * Formula: Y = 0.299*R + 0.587*G + 0.114*B
 *
 * This is the standard conversion used in image processing
 */
static uint8_t rgb_to_gray(uint8_t r, uint8_t g, uint8_t b) {
  // Using integer math for speed (scale by 1000)
  uint32_t gray = (299 * r + 587 * g + 114 * b) / 1000;
  return (uint8_t)gray;
}

/**
 * EXPORTED FUNCTION - Main grayscale converter
 *
 * Input: RGB image data (width * height * 3 bytes)
 * Output: Grayscale image data (width * height bytes)
 *
 * This is what your Flutter app will call
 */
__attribute__((visibility("default"))) __attribute__((used)) int
convert_to_grayscale(uint8_t *rgb_data, uint32_t width, uint32_t height,
                     uint8_t *output) {
  if (!rgb_data || !output || width == 0 || height == 0) {
    return -1; // Error: invalid input
  }

  uint32_t pixel_count = width * height;

  // Convert each pixel
  for (uint32_t i = 0; i < pixel_count; i++) {
    uint32_t rgb_index = i * 3;

    uint8_t r = rgb_data[rgb_index];
    uint8_t g = rgb_data[rgb_index + 1];
    uint8_t b = rgb_data[rgb_index + 2];

    output[i] = rgb_to_gray(r, g, b);
  }

  return 0; // Success
}

/**
 * ALTERNATIVE: Convert in-place (saves memory)
 * Converts RGB to grayscale and stores as RGB with equal R=G=B
 */
__attribute__((visibility("default"))) __attribute__((used)) void
convert_to_grayscale_inplace(uint8_t *rgb_data, uint32_t width,
                             uint32_t height) {
  uint32_t pixel_count = width * height;

  for (uint32_t i = 0; i < pixel_count; i++) {
    uint32_t index = i * 3;

    uint8_t r = rgb_data[index];
    uint8_t g = rgb_data[index + 1];
    uint8_t b = rgb_data[index + 2];

    uint8_t gray = rgb_to_gray(r, g, b);

    // Set all channels to gray value
    rgb_data[index] = gray;
    rgb_data[index + 1] = gray;
    rgb_data[index + 2] = gray;
  }
}

/**
 * HELPER: Get output size needed
 * Call this first to know how much memory to allocate
 */
__attribute__((visibility("default"))) __attribute__((used)) uint32_t
get_grayscale_size(uint32_t width, uint32_t height) {
  return width * height; // 1 byte per pixel for grayscale
}

/**
 * TEST FUNCTION: Simple verification
 * Converts a small test pattern
 */
__attribute__((visibility("default"))) __attribute__((used)) int
test_grayscale() {
  // 2x2 test image (RGB)
  uint8_t test_rgb[] = {
      255, 0,   0,   // Red pixel
      0,   255, 0,   // Green pixel
      0,   0,   255, // Blue pixel
      255, 255, 255  // White pixel
  };

  uint8_t output[4];

  int result = convert_to_grayscale(test_rgb, 2, 2, output);

  if (result != 0)
    return -1;

  // Verify results (approximate)
  // Red (255,0,0) -> 76
  // Green (0,255,0) -> 150
  // Blue (0,0,255) -> 29
  // White (255,255,255) -> 255

  if (output[0] >= 70 && output[0] <= 80 &&   // Red ~76
      output[1] >= 145 && output[1] <= 155 && // Green ~150
      output[2] >= 25 && output[2] <= 35 &&   // Blue ~29
      output[3] == 255) {                     // White
    return 1;                                 // Test passed!
  }

  return 0; // Test failed
}

/**
 * MAIN: Entry point for testing
 * When you run this WASM, it executes the test
 */
int main() {
  int result = test_grayscale();
  return result; // 1 = success, 0 = fail, -1 = error
}
