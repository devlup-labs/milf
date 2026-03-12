#include <stdint.h>
#include <stdio.h>
#include <string.h>

/* Helper to append strings safely */
static void append_str(uint8_t *buf, int *pos, const char *str) {
  int len = strlen(str);
  memcpy(buf + *pos, str, len);
  *pos += len;
}

/* Parse JPEG header to find width and height natively in C */
static int get_jpeg_dimensions(const uint8_t *data, int size, int *width,
                               int *height) {
  int pos = 0;
  // Check Start of Image (SOI) marker
  if (size < 2 || data[0] != 0xFF || data[1] != 0xD8)
    return 0;
  pos += 2;

  while (pos < size - 8) {
    if (data[pos] != 0xFF)
      return 0; // expected marker
    uint8_t marker = data[pos + 1];
    pos += 2;

    // SOF0, SOF1, SOF2 markers contain the dimensions
    if (marker == 0xC0 || marker == 0xC1 || marker == 0xC2) {
      pos += 3; // skip length and precision
      *height = (data[pos] << 8) | data[pos + 1];
      *width = (data[pos + 2] << 8) | data[pos + 3];
      return 1;
    }

    // Skip over other markers
    int len = (data[pos] << 8) | data[pos + 1];
    pos += len;
  }
  return 0; // Dimensions not found
}

/*
 * ABI: invoke(input_buffer, input_size, output_buffer, output_capacity)
 * Flutter sends a raw JPEG as `input_ptr`. We write PDF bytes to `output_ptr`.
 */
__attribute__((visibility("default"))) __attribute__((used)) int
invoke(uint8_t *input_ptr, int input_size, uint8_t *output_ptr,
       int output_capacity) {
  int width = 0, height = 0;

  // Extract image dimensions so the PDF page fits it perfectly
  if (!get_jpeg_dimensions(input_ptr, input_size, &width, &height)) {
    return -1; // Error: Not a valid JPEG
  }

  int pos = 0;
  char temp[256];

  // --- Header ---
  append_str(output_ptr, &pos, "%PDF-1.4\n");

  // --- Object 1: Catalog ---
  int obj1 = pos;
  append_str(output_ptr, &pos,
             "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n");

  // --- Object 2: Pages ---
  int obj2 = pos;
  append_str(output_ptr, &pos,
             "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n");

  // --- Object 3: Page --- (Sets page dimensions to match image)
  int obj3 = pos;
  snprintf(
      temp, sizeof(temp),
      "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources "
      "<< /XObject << /I1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n",
      width, height);
  append_str(output_ptr, &pos, temp);

  // --- Object 4: Image XObject --- (The crucial step that tells PDF to decode
  // a JPEG)
  int obj4 = pos;
  snprintf(temp, sizeof(temp),
           "4 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d "
           "/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode "
           "/Length %d >>\nstream\n",
           width, height, input_size);
  append_str(output_ptr, &pos, temp);

  // Embed the raw JPEG bytes directly into the PDF!
  memcpy(output_ptr + pos, input_ptr, input_size);
  pos += input_size;

  append_str(output_ptr, &pos, "\nendstream\nendobj\n");

  // --- Object 5: Content stream --- (Draws the image onto the page)
  int obj5 = pos;
  snprintf(temp, sizeof(temp), "q %d 0 0 %d 0 0 cm /I1 Do Q", width, height);
  int content_len = strlen(temp);

  char temp2[256];
  snprintf(temp2, sizeof(temp2),
           "5 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
           content_len, temp);
  append_str(output_ptr, &pos, temp2);

  // --- XRef table ---
  int xref = pos;
  append_str(output_ptr, &pos, "xref\n0 6\n0000000000 65535 f \n");
  snprintf(temp, sizeof(temp), "%010d 00000 n \n", obj1);
  append_str(output_ptr, &pos, temp);
  snprintf(temp, sizeof(temp), "%010d 00000 n \n", obj2);
  append_str(output_ptr, &pos, temp);
  snprintf(temp, sizeof(temp), "%010d 00000 n \n", obj3);
  append_str(output_ptr, &pos, temp);
  snprintf(temp, sizeof(temp), "%010d 00000 n \n", obj4);
  append_str(output_ptr, &pos, temp);
  snprintf(temp, sizeof(temp), "%010d 00000 n \n", obj5);
  append_str(output_ptr, &pos, temp);

  // --- Trailer ---
  snprintf(temp, sizeof(temp),
           "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%EOF\n", xref);
  append_str(output_ptr, &pos, temp);

  // Check if we didn't overflow memory
  if (pos > output_capacity)
    return -2;

  return pos; // Return number of bytes written to memory
}
