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
    snprintf(temp, sizeof(temp), "%d 0 R ", current_obj_id);
    append_str(output_ptr, &pos, temp);
    current_obj_id += 3; // Each page uses 3 objects (Page, Image, Content)
  }

  snprintf(temp, sizeof(temp), "] /Count %d >>\nendobj\n", num_images);
  append_str(output_ptr, &pos, temp);

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
    snprintf(temp, sizeof(temp),
             "%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] "
             "/Resources << /XObject << /I1 %d 0 R >> >> /Contents %d 0 R "
             ">>\nendobj\n",
             page_id, width, height, img_id, content_id);
    append_str(output_ptr, &pos, temp);

    // --- Object 2: Image XObject ---
    xref_offsets[img_id] = pos;
    snprintf(temp, sizeof(temp),
             "%d 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d "
             "/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode "
             "/Length %d >>\nstream\n",
             img_id, width, height, img_size);
    append_str(output_ptr, &pos, temp);

    // Embed JPEG bytes
    if (pos + img_size > output_capacity)
      return -2; // Check bounds
    memcpy(output_ptr + pos, img_data, img_size);
    pos += img_size;
    append_str(output_ptr, &pos, "\nendstream\nendobj\n");

    // --- Object 3: Content stream ---
    xref_offsets[content_id] = pos;
    snprintf(temp, sizeof(temp), "q %d 0 0 %d 0 0 cm /I1 Do Q", width, height);
    int content_len = strlen(temp);

    char temp2[256];
    snprintf(temp2, sizeof(temp2),
             "%d 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
             content_id, content_len, temp);
    append_str(output_ptr, &pos, temp2);
  }

  // --- XRef table ---
  int xref = pos;
  snprintf(temp, sizeof(temp), "xref\n0 %d\n0000000000 65535 f \n", obj_id);
  append_str(output_ptr, &pos, temp);

  for (int i = 1; i < obj_id; i++) {
    snprintf(temp, sizeof(temp), "%010d 00000 n \n", xref_offsets[i]);
    append_str(output_ptr, &pos, temp);
  }

  // --- Trailer ---
  snprintf(temp, sizeof(temp),
           "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%EOF\n",
           obj_id, xref);
  append_str(output_ptr, &pos, temp);

  if (pos > output_capacity)
    return -2;

  return pos;
}
