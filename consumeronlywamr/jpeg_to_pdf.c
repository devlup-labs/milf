#include <stdint.h>

/* --- Custom minimal libc replacements --- */
static void my_memcpy(uint8_t *dst, const uint8_t *src, int len) {
  for (int i = 0; i < len; i++) {
    dst[i] = src[i];
  }
}

static int my_strlen(const char *s) {
  int n = 0;
  while (s[n])
    n++;
  return n;
}

static void itoa_fixed(int val, char *buf, int width, int pad_zero) {
    if (val == 0) {
        if (width <= 0) width = 1;
        for (int i = 0; i < width - 1; i++) buf[i] = pad_zero ? '0' : ' ';
        buf[width - 1] = '0';
        buf[width] = '\0';
        return;
    }
    
    char temp[32];
    int i = 0;
    while (val > 0) {
        temp[i++] = '0' + (val % 10);
        val /= 10;
    }
    
    int padding = width - i;
    if (padding < 0) padding = 0;
    
    int b = 0;
    for (int p = 0; p < padding; p++) {
        buf[b++] = pad_zero ? '0' : ' ';
    }
    
    for (int j = i - 1; j >= 0; j--) {
        buf[b++] = temp[j];
    }
    buf[b] = '\0';
}

/* Helper to append strings safely */
static void append_str(uint8_t *buf, int *pos, const char *str) {
  int len = my_strlen(str);
  my_memcpy(buf + *pos, (const uint8_t*)str, len);
  *pos += len;
}

static void append_int(uint8_t *buf, int *pos, int val, int width, int pad_zero) {
    char temp[32];
    itoa_fixed(val, temp, width, pad_zero);
    append_str(buf, pos, temp);
}

static void append_obj_start(uint8_t *buf, int *pos, int id) {
    append_int(buf, pos, id, 0, 0);
    append_str(buf, pos, " 0 obj\n");
}
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
  append_obj_start(output_ptr, &pos, 3);
  append_str(output_ptr, &pos, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ");
  append_int(output_ptr, &pos, width, 0, 0);
  append_str(output_ptr, &pos, " ");
  append_int(output_ptr, &pos, height, 0, 0);
  append_str(output_ptr, &pos, "] /Resources << /XObject << /I1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n");

  // --- Object 4: Image XObject ---
  int obj4 = pos;
  append_obj_start(output_ptr, &pos, 4);
  append_str(output_ptr, &pos, "<< /Type /XObject /Subtype /Image /Width ");
  append_int(output_ptr, &pos, width, 0, 0);
  append_str(output_ptr, &pos, " /Height ");
  append_int(output_ptr, &pos, height, 0, 0);
  append_str(output_ptr, &pos, " /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ");
  append_int(output_ptr, &pos, input_size, 0, 0);
  append_str(output_ptr, &pos, " >>\nstream\n");

  // Embed the raw JPEG bytes directly into the PDF!
  my_memcpy(output_ptr + pos, input_ptr, input_size);
  pos += input_size;

  append_str(output_ptr, &pos, "\nendstream\nendobj\n");

  // --- Object 5: Content stream --- (Draws the image onto the page)
  int obj5 = pos;
  
  // Calculate content stream length manually: "q w 0 0 h 0 0 cm /I1 Do Q"
  char content_buf[64];
  int c_pos = 0;
  append_str((uint8_t*)content_buf, &c_pos, "q ");
  append_int((uint8_t*)content_buf, &c_pos, width, 0, 0);
  append_str((uint8_t*)content_buf, &c_pos, " 0 0 ");
  append_int((uint8_t*)content_buf, &c_pos, height, 0, 0);
  append_str((uint8_t*)content_buf, &c_pos, " 0 0 cm /I1 Do Q");
  int content_len = c_pos; // the length of the stream string

  append_obj_start(output_ptr, &pos, 5);
  append_str(output_ptr, &pos, "<< /Length ");
  append_int(output_ptr, &pos, content_len, 0, 0);
  append_str(output_ptr, &pos, " >>\nstream\n");
  
  // Directly copy the created content_buf
  my_memcpy(output_ptr + pos, (const uint8_t*)content_buf, content_len);
  pos += content_len;
  
  append_str(output_ptr, &pos, "\nendstream\nendobj\n");

  // --- XRef table ---
  int xref = pos;
  append_str(output_ptr, &pos, "xref\n0 6\n0000000000 65535 f \n");
  
  append_int(output_ptr, &pos, obj1, 10, 1); // 10-char wide, 0-padded
  append_str(output_ptr, &pos, " 00000 n \n");
  
  append_int(output_ptr, &pos, obj2, 10, 1);
  append_str(output_ptr, &pos, " 00000 n \n");
  
  append_int(output_ptr, &pos, obj3, 10, 1);
  append_str(output_ptr, &pos, " 00000 n \n");
  
  append_int(output_ptr, &pos, obj4, 10, 1);
  append_str(output_ptr, &pos, " 00000 n \n");
  
  append_int(output_ptr, &pos, obj5, 10, 1);
  append_str(output_ptr, &pos, " 00000 n \n");

  // --- Trailer ---
  append_str(output_ptr, &pos, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n");
  append_int(output_ptr, &pos, xref, 0, 0);
  append_str(output_ptr, &pos, "\n%%EOF\n");

  // Check if we didn't overflow memory
  if (pos > output_capacity)
    return -2;

  return pos; // Return number of bytes written to memory
}
