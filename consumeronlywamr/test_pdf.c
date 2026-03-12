#include <stdint.h>
#include <stdlib.h>

/* ============================================================
 * Minimal PDF generator — writes directly into WAMR linear memory.
 *
 * ABI:  int invoke(uint8_t *in, int in_sz, uint8_t *out, int out_cap)
 *       Returns the number of bytes written into `out`, or -1 on error.
 *
 * The host (JNI) allocates `out` inside WASM linear memory with
 * wasm_runtime_module_malloc, calls invoke(), then copies bytes back.
 * ============================================================ */

/* ---------- tiny helpers (no libc memcpy to avoid bulk-memory) ---------- */
static void my_copy(uint8_t *dst, const char *src, int len) {
  for (int i = 0; i < len; i++)
    dst[i] = (uint8_t)src[i];
}

static int my_strlen(const char *s) {
  int n = 0;
  while (s[n])
    n++;
  return n;
}

/* write a C string into buf, advance pos */
static void emit(uint8_t *buf, int *pos, const char *s) {
  int len = my_strlen(s);
  my_copy(buf + *pos, s, len);
  *pos += len;
}

/* ---------- PDF generation ---------- */
static int build_pdf(uint8_t *out, int cap) {
  int pos = 0;

  /* --- Header --- */
  emit(out, &pos, "%PDF-1.4\n");

  /* --- Object 1: Catalog --- */
  emit(out, &pos, "1 0 obj\n");
  emit(out, &pos, "<< /Type /Catalog /Pages 2 0 R >>\n");
  emit(out, &pos, "endobj\n\n");

  /* --- Object 2: Pages --- */
  emit(out, &pos, "2 0 obj\n");
  emit(out, &pos, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n");
  emit(out, &pos, "endobj\n\n");

  /* --- Object 3: Page --- */
  emit(out, &pos, "3 0 obj\n");
  emit(out, &pos, "<< /Type /Page /Parent 2 0 R\n");
  emit(out, &pos, "   /MediaBox [0 0 612 792]\n");
  emit(out, &pos, "   /Resources << /Font << /F1 5 0 R >> >>\n");
  emit(out, &pos, "   /Contents 4 0 R\n");
  emit(out, &pos, ">>\n");
  emit(out, &pos, "endobj\n\n");

  /* --- Object 4: Content stream --- */
  const char *drawing = "BT\n"
                        "/F1 24 Tf\n"
                        "100 700 Td\n"
                        "(Hello from WAMR!) Tj\n"
                        "0 -40 Td\n"
                        "(PDF generated inside WASM sandbox) Tj\n"
                        "ET\n";
  int drawing_len = my_strlen(drawing);

  /* stream length as decimal string */
  char len_str[16];
  {
    int v = drawing_len, i = 0;
    if (v == 0)
      len_str[i++] = '0';
    else {
      char tmp[16];
      int ti = 0;
      while (v > 0) {
        tmp[ti++] = '0' + (v % 10);
        v /= 10;
      }
      for (int j = ti - 1; j >= 0; j--)
        len_str[i++] = tmp[j];
    }
    len_str[i] = '\0';
  }

  emit(out, &pos, "4 0 obj\n");
  emit(out, &pos, "<< /Length ");
  emit(out, &pos, len_str);
  emit(out, &pos, " >>\n");
  emit(out, &pos, "stream\n");
  emit(out, &pos, drawing);
  emit(out, &pos, "endstream\n");
  emit(out, &pos, "endobj\n\n");

  /* --- Object 5: Font --- */
  emit(out, &pos, "5 0 obj\n");
  emit(out, &pos, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\n");
  emit(out, &pos, "endobj\n\n");

  /* --- Cross-reference table (simplified) --- */
  emit(out, &pos, "xref\n");
  emit(out, &pos, "0 6\n");
  emit(out, &pos, "0000000000 65535 f \n");
  emit(out, &pos, "0000000009 00000 n \n");
  emit(out, &pos, "0000000062 00000 n \n");
  emit(out, &pos, "0000000120 00000 n \n");
  emit(out, &pos, "0000000300 00000 n \n");
  emit(out, &pos, "0000000450 00000 n \n");

  /* --- Trailer --- */
  emit(out, &pos, "trailer\n");
  emit(out, &pos, "<< /Size 6 /Root 1 0 R >>\n");
  emit(out, &pos, "startxref\n");
  emit(out, &pos, "550\n");
  emit(out, &pos, "%%EOF\n");

  return pos;
}

/* ---------- Entry point (matches Universal Dispatcher ABI) ---------- */
__attribute__((visibility("default"))) __attribute__((used)) int
invoke(uint8_t *input_ptr, int input_size, uint8_t *output_ptr,
       int output_capacity) {

  int written = build_pdf(output_ptr, output_capacity);

  if (written <= 0 || written > output_capacity)
    return -1;

  return written;
}
