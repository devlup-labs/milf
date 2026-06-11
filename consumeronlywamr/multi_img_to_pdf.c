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
  if (size < 2 || data[0] != 0xFF || data[1] != 0xD8)
    return 0;
  pos += 2;

  while (pos < size - 8) {
    if (data[pos] != 0xFF)
      return 0; // expected marker
    uint8_t marker = data[pos + 1];
    pos += 2;

    if (marker == 0xC0 || marker == 0xC1 || marker == 0xC2) {
      pos += 3;
      *height = (data[pos] << 8) | data[pos + 1];
      *width = (data[pos + 2] << 8) | data[pos + 3];
      return 1;
    }

    int len = (data[pos] << 8) | data[pos + 1];
    pos += len;
  }
  return 0;
}

/*
 * ABI: invoke(input_buffer, input_size, output_buffer, output_capacity)
 *
 * Payload format:
 * [Number of Images: 4 bytes (Little Endian)]
 * For each image:
 *   [Size of Image: 4 bytes (Little Endian)]
 *   [Raw JPEG Bytes]
 */
__attribute__((visibility("default"))) __attribute__((used)) int
invoke(uint8_t *input_ptr, int input_size, uint8_t *output_ptr,
       int output_capacity) {
  if (input_size < 4)
    return -1;

  // Read number of images
  int num_images = input_ptr[0] | (input_ptr[1] << 8) | (input_ptr[2] << 16) |
                   (input_ptr[3] << 24);
  if (num_images <= 0 || num_images > 1000)
    return -1; // Sanity check

  int pos = 0;
  char temp[512];

  // Track object IDs
  int obj_id = 1;

  // --- Header ---
  append_str(output_ptr, &pos, "%PDF-1.4\n");

  // Arrays to hold object IDs for cross-reference
  int xref_offsets[3000]; // Max 3000 objects (supports ~1000 pages)
  xref_offsets[0] = 0;    // unused

  // --- Object 1: Catalog ---
  int catalog_id = obj_id++;
  xref_offsets[catalog_id] = pos;
  append_str(output_ptr, &pos,
             "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n");

  // --- Object 2: Pages ---
  int pages_id = obj_id++;
  xref_offsets[pages_id] = pos;

  // We need to list all Page objects inside the Kids array of the Pages object.
  // We know Page objects will be IDs 3, 6, 9... (if we use 3 objects per page).
  append_str(output_ptr, &pos, "2 0 obj\n<< /Type /Pages /Kids [");

  // Calculate the IDs of the Page objects beforehand
  int current_obj_id = pages_id + 1;
  for (int i = 0; i < num_images; i++) {
    append_int(output_ptr, &pos, current_obj_id, 0, 0);
    append_str(output_ptr, &pos, " 0 R ");
    current_obj_id += 3; // Each page uses 3 objects (Page, Image, Content)
  }

  append_str(output_ptr, &pos, "] /Count ");
  append_int(output_ptr, &pos, num_images, 0, 0);
  append_str(output_ptr, &pos, " >>\nendobj\n");

  // Parse payload and create pages
  int input_pos = 4;
  for (int i = 0; i < num_images; i++) {
    if (input_pos + 4 > input_size)
      return -1;

    // Read image size
    int img_size = input_ptr[input_pos] | (input_ptr[input_pos + 1] << 8) |
                   (input_ptr[input_pos + 2] << 16) |
                   (input_ptr[input_pos + 3] << 24);
    input_pos += 4;

    if (input_pos + img_size > input_size)
      return -1;

    // Read image bytes
    uint8_t *img_data = input_ptr + input_pos;
    input_pos += img_size;

    int width = 0, height = 0;
    if (!get_jpeg_dimensions(img_data, img_size, &width, &height)) {
      // Fallback default size if not a valid JPEG
      width = 600;
      height = 800;
    }

    // IDs for this page's objects
    int page_id = obj_id++;
    int img_id = obj_id++;
    int content_id = obj_id++;

    // --- Object 1: Page ---
    xref_offsets[page_id] = pos;
    append_obj_start(output_ptr, &pos, page_id);
    append_str(output_ptr, &pos, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ");
    append_int(output_ptr, &pos, width, 0, 0);
    append_str(output_ptr, &pos, " ");
    append_int(output_ptr, &pos, height, 0, 0);
    append_str(output_ptr, &pos, "] /Resources << /XObject << /I1 ");
    append_int(output_ptr, &pos, img_id, 0, 0);
    append_str(output_ptr, &pos, " 0 R >> >> /Contents ");
    append_int(output_ptr, &pos, content_id, 0, 0);
    append_str(output_ptr, &pos, " 0 R >>\nendobj\n");

    // --- Object 2: Image XObject ---
    xref_offsets[img_id] = pos;
    append_obj_start(output_ptr, &pos, img_id);
    append_str(output_ptr, &pos, "<< /Type /XObject /Subtype /Image /Width ");
    append_int(output_ptr, &pos, width, 0, 0);
    append_str(output_ptr, &pos, " /Height ");
    append_int(output_ptr, &pos, height, 0, 0);
    append_str(output_ptr, &pos, " /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ");
    append_int(output_ptr, &pos, img_size, 0, 0);
    append_str(output_ptr, &pos, " >>\nstream\n");

    // Embed JPEG bytes
    if (pos + img_size > output_capacity)
      return -2; // Check bounds
    my_memcpy(output_ptr + pos, img_data, img_size);
    pos += img_size;
    append_str(output_ptr, &pos, "\nendstream\nendobj\n");

    // --- Object 3: Content stream ---
    xref_offsets[content_id] = pos;
    
    char content_buf[64];
    int c_pos = 0;
    append_str((uint8_t*)content_buf, &c_pos, "q ");
    append_int((uint8_t*)content_buf, &c_pos, width, 0, 0);
    append_str((uint8_t*)content_buf, &c_pos, " 0 0 ");
    append_int((uint8_t*)content_buf, &c_pos, height, 0, 0);
    append_str((uint8_t*)content_buf, &c_pos, " 0 0 cm /I1 Do Q");
    int content_len = c_pos; // the length of the stream string

    append_obj_start(output_ptr, &pos, content_id);
    append_str(output_ptr, &pos, "<< /Length ");
    append_int(output_ptr, &pos, content_len, 0, 0);
    append_str(output_ptr, &pos, " >>\nstream\n");
    
    // Directly copy the created content_buf
    my_memcpy(output_ptr + pos, (const uint8_t*)content_buf, content_len);
    pos += content_len;
    
    append_str(output_ptr, &pos, "\nendstream\nendobj\n");
  }

  // --- XRef table ---
  int xref = pos;
  append_str(output_ptr, &pos, "xref\n0 ");
  append_int(output_ptr, &pos, obj_id, 0, 0);
  append_str(output_ptr, &pos, "\n0000000000 65535 f \n");

  for (int i = 1; i < obj_id; i++) {
    append_int(output_ptr, &pos, xref_offsets[i], 10, 1);
    append_str(output_ptr, &pos, " 00000 n \n");
  }

  // --- Trailer ---
  append_str(output_ptr, &pos, "trailer\n<< /Size ");
  append_int(output_ptr, &pos, obj_id, 0, 0);
  append_str(output_ptr, &pos, " /Root 1 0 R >>\nstartxref\n");
  append_int(output_ptr, &pos, xref, 0, 0);
  append_str(output_ptr, &pos, "\n%%EOF\n");

  if (pos > output_capacity)
    return -2;

  return pos;
}
