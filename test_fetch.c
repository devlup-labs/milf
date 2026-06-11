#include <milf.h>

void* malloc(unsigned long size);
void free(void* ptr);

// Helper to convert integer to string
static int itoa_simple(int val, char *buf) {
    if (val == 0) { buf[0] = '0'; return 1; }
    int temp = val, len = 0;
    while (temp > 0) { temp /= 10; len++; }
    for (int i = len - 1; i >= 0; i--) { buf[i] = '0' + (val % 10); val /= 10; }
    return len;
}

// Simple JPEG dimension parser
static int get_jpeg_dimensions(const unsigned char *data, int size, int *width, int *height) {
    if (size < 4 || data[0] != 0xFF || data[1] != 0xD8) return 0; // Not a JPEG
    int offset = 2;
    while (offset < size - 8) {
        if (data[offset] != 0xFF) return 0; // Invalid marker
        while (data[offset] == 0xFF) offset++; // Skip fill bytes
        unsigned char marker = data[offset++];
        int chunk_len = (data[offset] << 8) | data[offset+1];
        if (marker == 0xC0 || marker == 0xC1 || marker == 0xC2) {
            *height = (data[offset+3] << 8) | data[offset+4];
            *width = (data[offset+5] << 8) | data[offset+6];
            return 1;
        }
        offset += chunk_len;
    }
    return 0; // Dimensions not found
}

MILF_EXPORT int wasm_main(char *payload, int payload_len, char *out_buf, int out_max) {
    // 1. Fetch image from network instead of using payload
    // We use a GitHub raw JPEG URL because placeholder services often 301/302 redirect which may fail
    int handle = milf_stream_open("https://raw.githubusercontent.com/mdn/learning-area/master/html/multimedia-and-embedding/images-in-html/dinosaur_small.jpg");
    if (handle < 0) {
        milf_memcpy(out_buf, "Error: Failed to open image URL", 31);
        return 31;
    }

    // Allocate an initial buffer for the image
    int img_capacity = 65536; // 64KB initial
    char *img_data = (char *)malloc(img_capacity);
    int img_size = 0;
    
    // Read the image chunks
    while (1) {
        char chunk[4096];
        int bytes = milf_stream_read(handle, chunk, sizeof(chunk));
        if (bytes <= 0) break;
        
        // Expand buffer if needed
        if (img_size + bytes > img_capacity) {
            img_capacity *= 2;
            char *new_buf = (char *)malloc(img_capacity);
            milf_memcpy(new_buf, img_data, img_size);
            free(img_data);
            img_data = new_buf;
        }
        
        milf_memcpy(&img_data[img_size], chunk, bytes);
        img_size += bytes;
    }
    milf_stream_close(handle);

    if (img_size < 10) {
        free(img_data);
        milf_memcpy(out_buf, "Error: Image download failed", 28);
        return 28;
    }

    // Try to parse dimensions. Default to 800x600 if it fails (or if it's not a real jpeg but a png)
    int width = 800, height = 600;
    get_jpeg_dimensions((const unsigned char*)img_data, img_size, &width, &height);

    // Calculate dynamic offsets
    int obj1 = 0, obj2 = 0, obj3 = 0, obj4 = 0, obj5 = 0;
    
    // Allocate large buffer for PDF
    int pdf_capacity = img_size + 2048;
    char *pdf_buf = (char *)malloc(pdf_capacity);
    if (!pdf_buf) {
        free(img_data);
        milf_memcpy(out_buf, "Error: Malloc failed", 20);
        return 20;
    }
    
    int offset = 0;
    
    // PDF Header
    const char *header = "%PDF-1.4\n";
    milf_memcpy(&pdf_buf[offset], header, 9);
    offset += 9;

    // Object 1: Catalog
    obj1 = offset;
    const char *cat = "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n";
    milf_memcpy(&pdf_buf[offset], cat, 49);
    offset += 49;

    // Object 2: Pages
    obj2 = offset;
    const char *pages = "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n";
    milf_memcpy(&pdf_buf[offset], pages, 57);
    offset += 57;

    // Object 3: Page with matching MediaBox size
    obj3 = offset;
    milf_memcpy(&pdf_buf[offset], "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ", 52);
    offset += 52;
    offset += itoa_simple(width, &pdf_buf[offset]);
    pdf_buf[offset++] = ' ';
    offset += itoa_simple(height, &pdf_buf[offset]);
    const char *page_end = "] /Contents 4 0 R /Resources << /XObject << /I1 5 0 R >> >> >>\nendobj\n";
    milf_memcpy(&pdf_buf[offset], page_end, 71);
    offset += 71;

    // Object 4: Page Content (Draw Image)
    obj4 = offset;
    // content stream calculates matrix transformation
    char content_stream[128];
    int content_len = 0;
    content_stream[content_len++] = 'q';
    content_stream[content_len++] = '\n';
    content_len += itoa_simple(width, &content_stream[content_len]);
    content_stream[content_len++] = ' ';
    content_stream[content_len++] = '0';
    content_stream[content_len++] = ' ';
    content_stream[content_len++] = '0';
    content_stream[content_len++] = ' ';
    content_len += itoa_simple(height, &content_stream[content_len]);
    const char *cm = " 0 0 cm\n/I1 Do\nQ\n";
    milf_memcpy(&content_stream[content_len], cm, 17);
    content_len += 17;

    milf_memcpy(&pdf_buf[offset], "4 0 obj\n<< /Length ", 19);
    offset += 19;
    offset += itoa_simple(content_len, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " >>\nstream\n", 11);
    offset += 11;
    milf_memcpy(&pdf_buf[offset], content_stream, content_len);
    offset += content_len;
    milf_memcpy(&pdf_buf[offset], "endstream\nendobj\n", 18);
    offset += 18;

    // Object 5: Image XObject containing the JPEG bytes
    obj5 = offset;
    milf_memcpy(&pdf_buf[offset], "5 0 obj\n<< /Type /XObject /Subtype /Image /Width ", 49);
    offset += 49;
    offset += itoa_simple(width, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " /Height ", 9);
    offset += 9;
    offset += itoa_simple(height, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ", 73);
    offset += 73;
    offset += itoa_simple(img_size, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " >>\nstream\n", 11);
    offset += 11;
    
    // Write actual raw image bytes directly to PDF stream
    milf_memcpy(&pdf_buf[offset], img_data, img_size);
    offset += img_size;
    
    milf_memcpy(&pdf_buf[offset], "\nendstream\nendobj\n", 19);
    offset += 19;

    // XRef Table
    int xref_offset = offset;
    milf_memcpy(&pdf_buf[offset], "xref\n0 6\n0000000000 65535 f \n", 29);
    offset += 29;

    int locs[] = {obj1, obj2, obj3, obj4, obj5};
    for(int i=0; i<5; i++) {
        char loc_str[20];
        for(int j=0; j<10; j++) loc_str[j] = '0';
        int temp = locs[i], pos = 9;
        while(temp > 0) { loc_str[pos--] = '0' + (temp % 10); temp /= 10; }
        milf_memcpy(&pdf_buf[offset], loc_str, 10);
        offset += 10;
        milf_memcpy(&pdf_buf[offset], " 00000 n \n", 10);
        offset += 10;
    }

    // Trailer
    milf_memcpy(&pdf_buf[offset], "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n", 44);
    offset += 44;
    offset += itoa_simple(xref_offset, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], "\n%%EOF\n", 7);
    offset += 7;

    // Save to local storage
    int save_result = milf_storage_save("output_result.pdf", pdf_buf, offset);
    free(pdf_buf);
    free(img_data);

    if (save_result != 0) {
        milf_memcpy(out_buf, "Error: Storage save failed", 26);
        return 26;
    }

    const char *msg = "FILE:output_result.pdf";
    int msg_len = 23;
    milf_memcpy(out_buf, msg, msg_len);
    return msg_len;
}
