#include <mupdf/fitz.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

/**
 * Split PDF pages into 4 quadrants (Quads to Slides)
 *
 * ABI: int invoke(uint8_t *in, int in_sz, uint8_t *out, int out_cap)
 *
 * This uses MuPDF memory streams to avoid filesystem access,
 * keeping the security of our isolated process.
 */

__attribute__((visibility("default"))) __attribute__((used)) int
invoke(uint8_t *input_ptr, int input_size, uint8_t *output_ptr,
       int output_capacity) {
  fz_context *ctx = NULL;
  fz_document *doc = NULL;
  fz_output *out = NULL;
  fz_document_writer *writer = NULL;
  int written_size = -1;

  // 1. Initialize context
  ctx = fz_new_context(NULL, NULL, FZ_STORE_DEFAULT);
  if (!ctx)
    return -1;
  fz_register_document_handlers(ctx);

  fz_try(ctx) {
    // 2. Open input from memory stream
    fz_stream *stream = fz_open_memory(ctx, input_ptr, input_size);
    doc = fz_open_document_with_stream(ctx, "pdf", stream);
    fz_drop_stream(ctx, stream);

    int page_count = fz_count_pages(ctx, doc);

    // 3. Create output to memory stream
    // We use a buffer to collect the output bytes
    fz_buffer *buf = fz_new_buffer(ctx, 1024);
    out = fz_new_output_with_buffer(ctx, buf);
    writer = fz_new_document_writer(ctx, "pdf", "pdf", ""); // Empty options

    // 4. Process pages
    for (int i = 0; i < page_count; i++) {
      fz_page *page = fz_load_page(ctx, doc, i);
      fz_rect bounds;
      fz_bound_page(ctx, page, &bounds);

      float w = bounds.x1 - bounds.x0;
      float h = bounds.y1 - bounds.y0;
      float hw = w / 2;
      float hh = h / 2;

      fz_rect regions[4] = {
          {bounds.x0, bounds.y0, bounds.x0 + hw,
           bounds.y0 + hh}, // slide 1 (top-left)
          {bounds.x0 + hw, bounds.y0, bounds.x1,
           bounds.y0 + hh}, // slide 2 (top-right)
          {bounds.x0, bounds.y0 + hh, bounds.x0 + hw,
           bounds.y1}, // slide 3 (bottom-left)
          {bounds.x0 + hw, bounds.y0 + hh, bounds.x1,
           bounds.y1} // slide 4 (bottom-right)
      };

      for (int r = 0; r < 4; r++) {
        fz_device *dev;
        fz_rect mediabox = {0, 0, hw, hh}; // Each slide is half-size

        dev = fz_begin_page(ctx, writer, mediabox);

        // Shift the page content to show the specific quadrant
        fz_matrix ctm = fz_translate(-regions[r].x0, -regions[r].y0);
        fz_run_page(ctx, page, dev, &ctm, NULL);

        fz_end_page(ctx, writer);
      }
      fz_drop_page(ctx, page);
    }

    fz_close_document_writer(ctx, writer);

    // 5. Copy from buffer to output_ptr
    unsigned char *data;
    size_t len = fz_buffer_storage(ctx, buf, &data);

    if (len <= (size_t)output_capacity) {
      memcpy(output_ptr, data, len);
      written_size = (int)len;
    } else {
      written_size = -2; // Overflow
    }

    fz_drop_buffer(ctx, buf);
  }
  fz_always(ctx) {
    if (writer)
      fz_drop_document_writer(ctx, writer);
    if (out)
      fz_drop_output(ctx, out);
    if (doc)
      fz_drop_document(ctx, doc);
    fz_drop_context(ctx);
  }
  fz_catch(ctx) { written_size = -1; }

  return written_size;
}
