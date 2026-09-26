import { useState, useEffect, useMemo } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  Plus, Search, MoreVertical, Play, Trash2, Copy, ExternalLink, Loader2,
  Sparkles, BookOpen, Rocket, ChevronRight, Code2, Globe, FileText, Image,
  Terminal, Check, ArrowRight, Zap, Box, Network, GraduationCap, X, Info,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { AppLayout } from "@/components/layout";
import { PageHeader, DataTable, StatusBadge, EmptyState } from "@/components/shared";
import { GuidedTour, TourRestartButton } from "@/components/shared/GuidedTour";
import { DocsTab } from "@/components/shared/DocsTab";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useFunctions, useDeleteFunction, useInvokeFunction } from "@/hooks/useQueries";
import { FunctionEntity } from "@/lib/mock/types";
import { InvokeModal } from "@/components/InvokeModal";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

// ─────────────────────────────────────────────────────────────────────────────
// Demo Template definitions
// ─────────────────────────────────────────────────────────────────────────────

const SIMPLE_ADD_CODE = `#include <stdint.h>

__attribute__((visibility("default"))) __attribute__((used))
int wasm_main(int a, int b) {
  return a + b;
}`;

const ECHO_JSON_CODE = `#include <stdint.h>
#include <string.h>
#include <stdio.h>

// Export the function - this is what the WASM runtime will call.
// For complex data types (JSON/Strings), use this exact signature.
__attribute__((visibility("default"))) __attribute__((used))
int wasm_main(char* payload, int payload_len, char* out_buf, int out_max) {
  
  if (payload_len == 0) {
      const char* err = "{\\"error\\": \\"Empty payload\\"}";
      strncpy(out_buf, err, out_max);
      return strlen(err);
  }

  // Example: Echo the input back inside a JSON envelope
  int written = snprintf(out_buf, out_max, "{ \\"echo\\": %s, \\"status\\": \\"success\\" }", payload);
  return (written >= out_max) ? out_max - 1 : written;
}`;

const FETCH_TEXT_CODE = `#include <milf.h>

MILF_EXPORT int wasm_main(char* payload, int payload_len, char* out_buf, int out_max) {
    // 1. Open a stream to a public text file (We'll use a sample JSON endpoint)
    // You can replace this with any HTTPS URL!
    int handle = milf_stream_open("https://jsonplaceholder.typicode.com/todos/1");
    
    // Check for network errors or invalid URL
    if (handle < 0) {
        milf_memcpy(out_buf, "Error: Failed to open URL", 25);
        return 25;
    }

    // 2. Read the first chunk (up to what our target buffer supports)
    int bytes_read = milf_stream_read(handle, out_buf, out_max - 1);
    
    // Check for read errors
    if (bytes_read < 0) {
        milf_stream_close(handle);
        milf_memcpy(out_buf, "Error: Failed to read from stream", 34);
        return 34;
    }

    // 3. Null terminate the string safely
    out_buf[bytes_read] = '\\0';

    // 4. Close the network connection to avoid Memory/Socket leaks
    milf_stream_close(handle);

    // Return the amount of bytes we filled in the output buffer
    return bytes_read;
}`;

const URL_TO_PDF_CODE = `#include <milf.h>

// A minimal, structurally valid PDF. 
// We will replace "PLACEHOLDER" with the user's input payload.
static const char* pdf_template_part1 = 
    "%PDF-1.4\\n"
    "1 0 obj\\n"
    "<< /Type /Catalog /Pages 2 0 R >>\\n"
    "endobj\\n"
    "2 0 obj\\n"
    "<< /Type /Pages /Kids [3 0 R] /Count 1 >>\\n"
    "endobj\\n"
    "3 0 obj\\n"
    "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\\n"
    "endobj\\n"
    "4 0 obj\\n"
    "<< /Length %d >>\\n"
    "stream\\n"
    "BT\\n"
    "/F1 24 Tf\\n"
    "100 700 Td\\n"
    "(";

static const char* pdf_template_part2 = 
    ") Tj\\n"
    "ET\\n"
    "endstream\\n"
    "endobj\\n"
    "5 0 obj\\n"
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\\n"
    "endobj\\n"
    "xref\\n"
    "0 6\\n"
    "0000000000 65535 f \\n"
    "0000000009 00000 n \\n"
    "0000000058 00000 n \\n"
    "0000000115 00000 n \\n"
    "%010d 00000 n \\n"
    "%010d 00000 n \\n"
    "trailer\\n"
    "<< /Size 6 /Root 1 0 R >>\\n"
    "startxref\\n"
    "%d\\n"
    "%%EOF\\n";

MILF_EXPORT int wasm_main(char *payload, int payload_len, char *out_buf, int out_max) {
    // 1. Open a stream to a public text file (as done before)
    int handle = milf_stream_open("https://jsonplaceholder.typicode.com/todos/1");

    if (handle < 0) {
        milf_memcpy(out_buf, "Error: Failed to open URL", 25);
        return 25;
    }

    // Temporary buffer to hold the downloaded text
    char text_buf[4096];
    int bytes_read = milf_stream_read(handle, text_buf, sizeof(text_buf) - 1);
    milf_stream_close(handle);

    if (bytes_read < 0) {
        milf_memcpy(out_buf, "Error: Failed to read from stream", 34);
        return 34;
    }
    
    char text[4096];
    int text_len = 0;
    
    if (bytes_read > 0) {
        int max_len = bytes_read > 300 ? 300 : bytes_read;
        for (int i = 0; i < max_len; i++) {
            // Filter out parentheses and newlines to avoid breaking PDF syntax
            // PDF text strings cannot contain unescaped '(' or ')'
            if (text_buf[i] >= 32 && text_buf[i] <= 126 && text_buf[i] != '(' && text_buf[i] != ')') {
                text[text_len++] = text_buf[i];
            }
        }
    }
    
    if (text_len == 0) {
        text_len = 11;
        milf_memcpy(text, "Hello World", text_len);
    }
    text[text_len] = '\\0';

    // Calculate length of the stream dictionary text
    // "BT\\n/F1 24 Tf\\n100 700 Td\\n(TEXT) Tj\\nET\\n"
    int stream_length = 3 + 12 + 11 + 1 + text_len + 5 + 3;

    // Build the PDF in memory
    char pdf_buf[8192];
    int offset = 0;

    // Format part 1 carefully avoiding standard snprintf which might be missing in WASM
    // We'll calculate offsets manually to ensure valid cross-reference tables
    
    // Part 1 length (static) = 239 bytes with %d replaced
    char part1_buf[512];
    int part1_len = 0;
    
    // Simple integer to string for length
    char len_str[16];
    int len_val = stream_length;
    int len_digits = 0;
    do {
        len_str[len_digits++] = '0' + (len_val % 10);
        len_val /= 10;
    } while (len_val > 0 && len_digits < 15);
    
    const char* p = pdf_template_part1;
    while (*p) {
        if (*p == '%' && *(p+1) == 'd') {
            for (int i = len_digits - 1; i >= 0; i--) {
                part1_buf[part1_len++] = len_str[i];
            }
            p += 2;
        } else {
            part1_buf[part1_len++] = *p++;
        }
    }
    part1_buf[part1_len] = '\\0';

    // Copy part1 to main buffer
    milf_memcpy(&pdf_buf[offset], part1_buf, part1_len);
    offset += part1_len;

    // Copy text payload
    milf_memcpy(&pdf_buf[offset], text, text_len);
    offset += text_len;

    // Calculate obj 4 offset (before adding part 2)
    int obj4_offset = 237; 

    const char* static_part2a = 
        ") Tj\\n"
        "ET\\n"
        "endstream\\n"
        "endobj\\n";
    int p2a_len = 26;
    
    milf_memcpy(&pdf_buf[offset], static_part2a, p2a_len);
    offset += p2a_len;

    int obj5_offset = offset;

    const char* static_part2b = 
        "5 0 obj\\n"
        "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\\n"
        "endobj\\n";
    int p2b_len = 66;

    milf_memcpy(&pdf_buf[offset], static_part2b, p2b_len);
    offset += p2b_len;

    int xref_offset = offset;

    const char* static_part2c = 
        "xref\\n"
        "0 6\\n"
        "0000000000 65535 f \\n"
        "0000000009 00000 n \\n"
        "0000000058 00000 n \\n"
        "0000000115 00000 n \\n";
    int p2c_len = 101;

    milf_memcpy(&pdf_buf[offset], static_part2c, p2c_len);
    offset += p2c_len;

    // Write obj 4 and 5 offsets (10 chars each)
    char obj4_str[20];
    for (int i = 0; i < 10; i++) obj4_str[i] = '0';
    int temp_val = obj4_offset, pos = 9;
    while(temp_val > 0) { obj4_str[pos--] = '0' + (temp_val % 10); temp_val /= 10; }
    milf_memcpy(&pdf_buf[offset], obj4_str, 10);
    offset += 10;
    
    milf_memcpy(&pdf_buf[offset], " 00000 n \\n", 10);
    offset += 10;

    char obj5_str[20];
    for (int i = 0; i < 10; i++) obj5_str[i] = '0';
    temp_val = obj5_offset; pos = 9;
    while(temp_val > 0) { obj5_str[pos--] = '0' + (temp_val % 10); temp_val /= 10; }
    milf_memcpy(&pdf_buf[offset], obj5_str, 10);
    offset += 10;
    
    milf_memcpy(&pdf_buf[offset], " 00000 n \\n", 10);
    offset += 10;

    const char* static_part2d = 
        "trailer\\n"
        "<< /Size 6 /Root 1 0 R >>\\n"
        "startxref\\n";
    int p2d_len = 45;

    milf_memcpy(&pdf_buf[offset], static_part2d, p2d_len);
    offset += p2d_len;

    // Write xref offset
    char xref_str[16];
    temp_val = xref_offset; pos = 0;
    do {
        xref_str[pos++] = '0' + (temp_val % 10);
        temp_val /= 10;
    } while (temp_val > 0);
    
    // Reverse xref str
    for (int i = 0; i < pos / 2; i++) {
        char t = xref_str[i];
        xref_str[i] = xref_str[pos - i - 1];
        xref_str[pos - i - 1] = t;
    }
    
    milf_memcpy(&pdf_buf[offset], xref_str, pos);
    offset += pos;

    const char* static_part2e = "\\n%%EOF\\n";
    milf_memcpy(&pdf_buf[offset], static_part2e, 7);
    offset += 7;

    // Save PDF to local storage
    int save_result = milf_storage_save("output_result.pdf", pdf_buf, offset);

    if (save_result != 0) {
        milf_memcpy(out_buf, "Error: Storage save failed", 26);
        return 26;
    }

    const char *msg = "FILE:output_result.pdf";
    int msg_len = 23;
    milf_memcpy(out_buf, msg, msg_len);
    return msg_len;
}`;

const IMAGE_TO_PDF_CODE = `#include <milf.h>

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
    int handle = milf_stream_open("https://upload.wikimedia.org/wikipedia/commons/f/f9/Phoenicopterus_ruber_in_S%C3%A3o_Paulo_Zoo.jpg");
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
    const char *header = "%PDF-1.4\\n";
    milf_memcpy(&pdf_buf[offset], header, 9);
    offset += 9;

    // Object 1: Catalog
    obj1 = offset;
    const char *cat = "1 0 obj\\n<< /Type /Catalog /Pages 2 0 R >>\\nendobj\\n";
    milf_memcpy(&pdf_buf[offset], cat, 49);
    offset += 49;

    // Object 2: Pages
    obj2 = offset;
    const char *pages = "2 0 obj\\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\\nendobj\\n";
    milf_memcpy(&pdf_buf[offset], pages, 57);
    offset += 57;

    // Object 3: Page with matching MediaBox size
    obj3 = offset;
    milf_memcpy(&pdf_buf[offset], "3 0 obj\\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ", 52);
    offset += 52;
    offset += itoa_simple(width, &pdf_buf[offset]);
    pdf_buf[offset++] = ' ';
    offset += itoa_simple(height, &pdf_buf[offset]);
    const char *page_end = "] /Contents 4 0 R /Resources << /XObject << /I1 5 0 R >> >> >>\\nendobj\\n";
    milf_memcpy(&pdf_buf[offset], page_end, 71);
    offset += 71;

    // Object 4: Page Content (Draw Image)
    obj4 = offset;
    // content stream calculates matrix transformation
    char content_stream[128];
    int content_len = 0;
    content_stream[content_len++] = 'q';
    content_stream[content_len++] = '\\n';
    content_len += itoa_simple(width, &content_stream[content_len]);
    content_stream[content_len++] = ' ';
    content_stream[content_len++] = '0';
    content_stream[content_len++] = ' ';
    content_stream[content_len++] = '0';
    content_stream[content_len++] = ' ';
    content_len += itoa_simple(height, &content_stream[content_len]);
    const char *cm = " 0 0 cm\\n/I1 Do\\nQ\\n";
    milf_memcpy(&content_stream[content_len], cm, 17);
    content_len += 17;

    milf_memcpy(&pdf_buf[offset], "4 0 obj\\n<< /Length ", 19);
    offset += 19;
    offset += itoa_simple(content_len, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " >>\\nstream\\n", 11);
    offset += 11;
    milf_memcpy(&pdf_buf[offset], content_stream, content_len);
    offset += content_len;
    milf_memcpy(&pdf_buf[offset], "endstream\\nendobj\\n", 18);
    offset += 18;

    // Object 5: Image XObject containing the JPEG bytes
    obj5 = offset;
    milf_memcpy(&pdf_buf[offset], "5 0 obj\\n<< /Type /XObject /Subtype /Image /Width ", 49);
    offset += 49;
    offset += itoa_simple(width, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " /Height ", 9);
    offset += 9;
    offset += itoa_simple(height, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ", 73);
    offset += 73;
    offset += itoa_simple(img_size, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], " >>\\nstream\\n", 11);
    offset += 11;
    
    // Write actual raw image bytes directly to PDF stream
    milf_memcpy(&pdf_buf[offset], img_data, img_size);
    offset += img_size;
    
    milf_memcpy(&pdf_buf[offset], "\\nendstream\\nendobj\\n", 19);
    offset += 19;

    // XRef Table
    int xref_offset = offset;
    milf_memcpy(&pdf_buf[offset], "xref\\n0 6\\n0000000000 65535 f \\n", 29);
    offset += 29;

    int locs[] = {obj1, obj2, obj3, obj4, obj5};
    for(int i=0; i<5; i++) {
        char loc_str[20];
        for(int j=0; j<10; j++) loc_str[j] = '0';
        int temp = locs[i], pos = 9;
        while(temp > 0) { loc_str[pos--] = '0' + (temp % 10); temp /= 10; }
        milf_memcpy(&pdf_buf[offset], loc_str, 10);
        offset += 10;
        milf_memcpy(&pdf_buf[offset], " 00000 n \\n", 10);
        offset += 10;
    }

    // Trailer
    milf_memcpy(&pdf_buf[offset], "trailer\\n<< /Size 6 /Root 1 0 R >>\\nstartxref\\n", 44);
    offset += 44;
    offset += itoa_simple(xref_offset, &pdf_buf[offset]);
    milf_memcpy(&pdf_buf[offset], "\\n%%EOF\\n", 7);
    offset += 7;

    // Save to local storage
    int save_result = milf_storage_save("output_result.pdf", pdf_buf, offset);
    free(pdf_buf);
    free(img_data);

    const char *msg = "FILE:output_result.pdf";
    milf_memcpy(out_buf, msg, 23);
    return 23;
}`;

const MULTI_IMAGES_TO_PDF_CODE = `#include <milf.h>

// Up to 5 image URLs, each up to 256 chars
#define MAX_URL_COUNT 5
#define MAX_URL_LENGTH 256

static char* find_sub(char* h, int hl, const char* n) {
    int nl = milf_strlen(n);
    if (!nl || hl < nl) return NULL;
    for (int i = 0; i <= hl - nl; ++i)
        if (milf_memcmp(h + i, n, nl) == 0) return h + i;
    return NULL;
}

// 3MB bump allocator — no malloc needed
static char g_pool[1024 * 1024 * 3];
static int  g_pool_off = 0;
static void* pa(int sz) {
    int a = (sz + 7) & ~7;
    if (g_pool_off + a > (int)sizeof(g_pool)) return NULL;
    void* p = &g_pool[g_pool_off]; g_pool_off += a; return p;
}

static char g_urls[MAX_URL_COUNT][MAX_URL_LENGTH];
static int  g_nurls = 0;

struct Img { char* data; int size, width, height; };
static struct Img g_imgs[MAX_URL_COUNT];
static int g_nimgs = 0;

static int jpeg_dim(const unsigned char* d, int sz, int* w, int* h) {
    if (sz < 4 || d[0] != 0xFF || d[1] != 0xD8) return 0;
    int o = 2;
    while (o < sz - 8) {
        if (d[o] != 0xFF) return 0;
        while (d[o] == 0xFF) o++;
        unsigned char m = d[o++];
        int cl = (d[o] << 8) | d[o+1];
        if (m == 0xC0 || m == 0xC1 || m == 0xC2) {
            *h = (d[o+3]<<8)|d[o+4]; *w = (d[o+5]<<8)|d[o+6]; return 1;
        }
        o += cl;
    }
    return 0;
}

static int as(char* b,int* o,int m,const char* s){int l=milf_strlen(s);if(*o+l>=m)return 0;milf_memcpy(b+*o,s,l);*o+=l;return 1;}
static int ai(char* b,int* o,int m,int v){
    char t[12];milf_memset(t,0,12);
    if(v==0){t[0]='0';}else{unsigned u=(unsigned)v;int i=0;while(u){t[i++]='0'+(u%10);u/=10;}for(int a=0,z=i-1;a<z;a++,z--){char c=t[a];t[a]=t[z];t[z]=c;}}
    int l=milf_strlen(t);if(*o+l>=m)return 0;milf_memcpy(b+*o,t,l);*o+=l;return 1;
}
static int ab(char* b,int* o,int m,const char* d,int l){if(*o+l>=m)return 0;milf_memcpy(b+*o,d,l);*o+=l;return 1;}
static void xr(int off,char* dst){
    for(int j=0;j<10;j++)dst[j]='0';
    int p=9;while(off>0){dst[p--]='0'+(off%10);off/=10;}
}

MILF_EXPORT int wasm_main(char* payload, int payload_len, char* out_buf, int out_max) {
    g_nurls=0; g_nimgs=0; g_pool_off=0;
    milf_memset(g_urls,0,sizeof(g_urls));

    char* p=payload, *end=payload+payload_len;

    // Parse {"image_urls":["url1","url2",...]}
    char* mk = find_sub(p,payload_len,"image_urls");
    if (!mk) { const char* e="{\\"error\\":\\"Missing image_urls\\"}"; milf_memcpy(out_buf,e,milf_strlen(e)); return milf_strlen(e); }
    p = mk + milf_strlen("image_urls");
    char* br = find_sub(p,end-p,"[");
    if (!br) { const char* e="{\\"error\\":\\"No array\\"}"; milf_memcpy(out_buf,e,milf_strlen(e)); return milf_strlen(e); }
    p = br + 1;

    while (p < end && g_nurls < MAX_URL_COUNT) {
        while (p<end&&(*p==' '||*p=='\\n'||*p=='\\r'||*p=='\\t'||*p==','||*p==':')) p++;
        if (p>=end||*p==']') break;
        char* qs=find_sub(p,end-p,"\\""); if(!qs) break; qs++;
        char* qe=find_sub(qs,end-qs,"\\""); if(!qe) break;
        int ul=qe-qs;
        if(ul>0&&ul<MAX_URL_LENGTH){milf_memcpy(g_urls[g_nurls],qs,ul);g_urls[g_nurls][ul]='\\0';g_nurls++;}
        p=qe+1;
    }

    if (!g_nurls) { const char* e="{\\"error\\":\\"No URLs\\"}"; milf_memcpy(out_buf,e,milf_strlen(e)); return milf_strlen(e); }

    // Fetch images
    for (int i=0;i<g_nurls;i++) {
        char* buf=(char*)pa(512*1024); if(!buf) continue;
        int h=milf_stream_open(g_urls[i]); if(h<0) continue;
        int tot=0;
        while(tot<512*1024){int b=milf_stream_read(h,buf+tot,4096);if(b<=0)break;tot+=b;}
        milf_stream_close(h);
        if(tot>10){int w=800,ht=600;jpeg_dim((const unsigned char*)buf,tot,&w,&ht);
            g_imgs[g_nimgs++]=(struct Img){buf,tot,w,ht};}
    }

    if(!g_nimgs){const char* e="{\\"error\\":\\"No images fetched\\"}";milf_memcpy(out_buf,e,milf_strlen(e));return milf_strlen(e);}

    // Build multi-page PDF
    int off=0;
    int ofs[30]; milf_memset(ofs,0,sizeof(ofs));

    as(out_buf,&off,out_max,"%PDF-1.4\\n");
    ofs[1]=off; as(out_buf,&off,out_max,"1 0 obj\\n<< /Type /Catalog /Pages 2 0 R >>\\nendobj\\n");
    ofs[2]=off; as(out_buf,&off,out_max,"2 0 obj\\n<< /Type /Pages /Kids [");
    for(int i=0;i<g_nimgs;i++){ai(out_buf,&off,out_max,3+3*i);as(out_buf,&off,out_max," 0 R ");}
    as(out_buf,&off,out_max,"] /Count ");ai(out_buf,&off,out_max,g_nimgs);as(out_buf,&off,out_max," >>\\nendobj\\n");

    for(int i=0;i<g_nimgs;i++){
        // Page object
        ofs[3+3*i]=off;
        ai(out_buf,&off,out_max,3+3*i);as(out_buf,&off,out_max," 0 obj\\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ");
        ai(out_buf,&off,out_max,g_imgs[i].width);as(out_buf,&off,out_max," ");ai(out_buf,&off,out_max,g_imgs[i].height);
        as(out_buf,&off,out_max,"] /Contents ");ai(out_buf,&off,out_max,3+3*i+1);
        as(out_buf,&off,out_max," 0 R /Resources << /XObject << /I");ai(out_buf,&off,out_max,i+1);
        as(out_buf,&off,out_max," ");ai(out_buf,&off,out_max,3+3*i+2);as(out_buf,&off,out_max," 0 R >> >> >>\\nendobj\\n");
        // Content stream: draw image full-page
        char cs[128]; milf_memset(cs,0,128); int cl=0;
        cs[cl++]='q';cs[cl++]='\\n';
        char ws[12];milf_memset(ws,0,12);int wv=g_imgs[i].width,wi=0;
        if(!wv){ws[wi++]='0';}else{unsigned u=(unsigned)wv;while(u){ws[wi++]='0'+(u%10);u/=10;}for(int a=0,z=wi-1;a<z;a++,z--){char t=ws[a];ws[a]=ws[z];ws[z]=t;}}
        milf_memcpy(cs+cl,ws,milf_strlen(ws));cl+=milf_strlen(ws);
        milf_memcpy(cs+cl," 0 0 ",5);cl+=5;
        char hs[12];milf_memset(hs,0,12);int hv=g_imgs[i].height,hi=0;
        if(!hv){hs[hi++]='0';}else{unsigned u=(unsigned)hv;while(u){hs[hi++]='0'+(u%10);u/=10;}for(int a=0,z=hi-1;a<z;a++,z--){char t=hs[a];hs[a]=hs[z];hs[z]=t;}}
        milf_memcpy(cs+cl,hs,milf_strlen(hs));cl+=milf_strlen(hs);
        milf_memcpy(cs+cl," 0 0 cm\\n/I",10);cl+=10;
        char is2[12];milf_memset(is2,0,12);int iv=i+1,ii=0;
        if(!iv){is2[ii++]='0';}else{unsigned u=(unsigned)iv;while(u){is2[ii++]='0'+(u%10);u/=10;}for(int a=0,z=ii-1;a<z;a++,z--){char t=is2[a];is2[a]=is2[z];is2[z]=t;}}
        milf_memcpy(cs+cl,is2,milf_strlen(is2));cl+=milf_strlen(is2);
        milf_memcpy(cs+cl," Do\\nQ\\n",6);cl+=6;
        ofs[3+3*i+1]=off;
        ai(out_buf,&off,out_max,3+3*i+1);as(out_buf,&off,out_max," 0 obj\\n<< /Length ");ai(out_buf,&off,out_max,cl);
        as(out_buf,&off,out_max," >>\\nstream\\n");ab(out_buf,&off,out_max,cs,cl);as(out_buf,&off,out_max,"\\nendstream\\nendobj\\n");
        // Image XObject (raw JPEG bytes via DCTDecode)
        ofs[3+3*i+2]=off;
        ai(out_buf,&off,out_max,3+3*i+2);as(out_buf,&off,out_max," 0 obj\\n<< /Type /XObject /Subtype /Image /Width ");
        ai(out_buf,&off,out_max,g_imgs[i].width);as(out_buf,&off,out_max," /Height ");ai(out_buf,&off,out_max,g_imgs[i].height);
        as(out_buf,&off,out_max," /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ");
        ai(out_buf,&off,out_max,g_imgs[i].size);as(out_buf,&off,out_max," >>\\nstream\\n");
        ab(out_buf,&off,out_max,g_imgs[i].data,g_imgs[i].size);as(out_buf,&off,out_max,"\\nendstream\\nendobj\\n");
    }

    // XRef table
    int xref_off=off;
    int total_obj=2+3*g_nimgs;
    as(out_buf,&off,out_max,"xref\\n0 ");ai(out_buf,&off,out_max,total_obj+1);as(out_buf,&off,out_max,"\\n0000000000 65535 f \\n");
    for(int k=1;k<=total_obj;k++){
        char e[21];milf_memset(e,0,21);xr(ofs[k],e);milf_memcpy(e+10," 00000 n \\n",10);ab(out_buf,&off,out_max,e,20);
    }
    as(out_buf,&off,out_max,"trailer\\n<< /Size ");ai(out_buf,&off,out_max,total_obj+1);
    as(out_buf,&off,out_max," /Root 1 0 R >>\\nstartxref\\n");ai(out_buf,&off,out_max,xref_off);as(out_buf,&off,out_max,"\\n%%EOF\\n");

    milf_storage_save("output.pdf", out_buf, off);
    const char* ref="FILE:output.pdf"; int rl=milf_strlen(ref);
    milf_memcpy(out_buf,ref,rl); out_buf[rl]='\\0'; return rl;
}`;

interface DemoTemplate {
  id: string;
  name: string;
  tagline: string;
  description: string;
  difficulty: "Beginner" | "Intermediate" | "Advanced";
  icon: React.ElementType;
  iconColor: string;
  iconBg: string;
  badges: string[];
  code: string;
  runtime: string;
  sampleInput: string;
  expectedOutput: string;
  abiExplanation?: string;
}

const DEMO_TEMPLATES: DemoTemplate[] = [
  {
    id: "simple-add",
    name: "simple-add",
    tagline: "Your first WASM function",
    description:
      "The simplest possible WASM function. Takes two integers and returns their sum. No headers needed — pure compute at < 5ms cold start.",
    difficulty: "Beginner",
    icon: Zap,
    iconColor: "text-yellow-400",
    iconBg: "bg-yellow-400/10 border-yellow-400/20",
    badges: ["C", "Primitive types"],
    code: SIMPLE_ADD_CODE,
    runtime: "c",
    sampleInput: `{"a": 3, "b": 7}`,
    expectedOutput: `10`,
  },
  {
    id: "echo-json",
    name: "echo-json",
    tagline: "JSON in, JSON out",
    description:
      "Accepts any JSON payload and echoes it back wrapped in a success envelope. Great starting point for building real data-processing functions.",
    difficulty: "Beginner",
    icon: Code2,
    iconColor: "text-blue-400",
    iconBg: "bg-blue-400/10 border-blue-400/20",
    badges: ["C", "JSON", "String I/O"],
    code: ECHO_JSON_CODE,
    runtime: "c",
    sampleInput: `{\n  "type": "json",\n  "data": {\n    "_func": "wasm_main",\n    "payload": "{\\"message\\": \\"hello from the edge\\"}",\n    "payload_len": 32,\n    "out_buf": "",\n    "out_max": 256\n  }\n}`,
    expectedOutput: `{ "echo": {"message": "hello from the edge"}, "status": "success" }`,
  },
  {
    id: "fetch-text",
    name: "fetch-text",
    tagline: "Fetch live data from the internet",
    description:
      "Uses the milf.h networking API to open an HTTP stream, read the response, and return the raw text. No curl, no stdlib — just milf_stream_*.",
    difficulty: "Intermediate",
    icon: Globe,
    iconColor: "text-emerald-400",
    iconBg: "bg-emerald-400/10 border-emerald-400/20",
    badges: ["C", "milf.h", "Network", "Streaming"],
    code: FETCH_TEXT_CODE,
    runtime: "c",
    sampleInput: `{\n  "type": "json",\n  "data": {\n    "_func": "wasm_main",\n    "payload": "",\n    "payload_len": 0,\n    "out_buf": "",\n    "out_max": 512\n  }\n}`,
    expectedOutput: `{\n  "userId": 1,\n  "id": 1,\n  "title": "delectus aut autem",\n  "completed": false\n}`,
  },
  {
    id: "url-to-pdf",
    name: "url-to-pdf",
    tagline: "Render live data as a PDF",
    description:
      "Fetches JSON from the web, strips unsafe characters, and constructs a structurally valid PDF from scratch — entirely inside WASM. The result is saved to local storage and returned as a download link.",
    difficulty: "Advanced",
    icon: FileText,
    iconColor: "text-rose-400",
    iconBg: "bg-rose-400/10 border-rose-400/20",
    badges: ["C", "milf.h", "Network", "PDF", "Storage"],
    code: URL_TO_PDF_CODE,
    runtime: "c",
    sampleInput: `{\n  "type": "json",\n  "data": {\n    "_func": "wasm_main",\n    "payload": "",\n    "payload_len": 0,\n    "out_buf": "",\n    "out_max": 10240\n  }\n}`,
    expectedOutput: `FILE:output_result.pdf\n→ Triggers a PDF download in the dashboard`,
  },
  {
    id: "image-to-pdf",
    name: "image-to-pdf",
    tagline: "Download a JPEG and wrap it in a PDF",
    description:
      "Downloads a full JPEG image over HTTP, parses its dimensions, and embeds the raw DCT-encoded bytes directly into a PDF XObject stream. Uses dynamic memory with malloc/free.",
    difficulty: "Advanced",
    icon: Image,
    iconColor: "text-purple-400",
    iconBg: "bg-purple-400/10 border-purple-400/20",
    badges: ["C", "milf.h", "malloc", "Network", "PDF", "JPEG"],
    code: IMAGE_TO_PDF_CODE,
    runtime: "c",
    sampleInput: `{\n  "type": "json",\n  "data": {\n    "_func": "wasm_main",\n    "payload": "",\n    "payload_len": 0,\n    "out_buf": "",\n    "out_max": 10240\n  }\n}`,
    expectedOutput: `FILE:output_result.pdf\n→ PDF containing the full-size flamingo photo`,
  },
  {
    id: "multi-images-to-pdf",
    name: "multi-images-to-pdf",
    tagline: "Batch-convert a list of image URLs into one multi-page PDF",
    description:
      "Accepts a JSON array of up to 5 JPEG URLs, fetches each one via milf_stream_*, parses JPEG dimensions, and assembles a structurally valid multi-page PDF — one image per page — using a 3MB bump allocator instead of malloc. No standard library required.",
    difficulty: "Advanced",
    icon: FileText,
    iconColor: "text-orange-400",
    iconBg: "bg-orange-400/10 border-orange-400/20",
    badges: ["C", "milf.h", "Network", "PDF", "Multi-page", "Bump allocator", "JPEG"],
    code: MULTI_IMAGES_TO_PDF_CODE,
    runtime: "c",
    sampleInput: `{\n  "type": "json",\n  "data": {\n    "_func": "wasm_main",\n    "payload": "{\\"image_urls\\": [\\"https://picsum.photos/id/237/400/300.jpg\\", \\"https://picsum.photos/id/1015/400/300.jpg\\",\\"https://http.cat/200.jpg\\",\\"https://picsum.photos/id/1025/400/300.jpg\\",\\"https://picsum.photos/id/1074/400/300.jpg\\"]}",\n    "payload_len": 226,\n    "out_buf": "",\n    "out_max": 256\n  }\n}`,
    expectedOutput: `FILE:output.pdf\n→ 2-page PDF, one JPEG per page, sized to each image's native resolution`,
    abiExplanation: "1. payload: Pointer to the JSON input array of URLs.\n2. payload_len: Length of the JSON string (226 bytes).\n3. out_buf: Pre-allocated buffer where the generated PDF bytes are written.\n4. out_max: Maximum capacity of the output buffer to prevent memory corruption.",
  },
];

const DIFFICULTY_STYLES: Record<string, string> = {
  Beginner: "bg-green-500/10 text-green-400 border border-green-500/20",
  Intermediate: "bg-yellow-500/10 text-yellow-400 border border-yellow-500/20",
  Advanced: "bg-rose-500/10 text-rose-400 border border-rose-500/20",
};

// ─────────────────────────────────────────────────────────────────────────────
// DemoTemplateCard
// ─────────────────────────────────────────────────────────────────────────────
const CCodeViewer = ({ code, expanded, setExpanded }: { code: string; expanded: boolean; setExpanded: (v: boolean) => void }) => {
  const highlightedCodeHtml = useMemo(() => {
    // Escape HTML
    let html = code
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");

    // Highlight comments (both // and /* */)
    const comments: string[] = [];
    html = html.replace(/(\/\/.*|\/\*[\s\S]*?\*\/)/g, (match) => {
      comments.push(match);
      return `___COMMENT_PLACEHOLDER_${comments.length - 1}___`;
    });

    // Highlight strings
    html = html.replace(/("[^"\\]*(?:\\.[^"\\]*)*")/g, '<span class="text-yellow-300">$1</span>');

    // Highlight preprocessor directives
    html = html.replace(/(#include\s+&lt;.*?&gt;|#include\s+".*?")/g, '<span class="text-rose-400 font-semibold">$1</span>');
    html = html.replace(/(#define\s+\w+)/g, '<span class="text-rose-400 font-semibold">$1</span>');

    // Highlight types and keywords
    const keywords = ["int", "char", "void", "return", "if", "else", "while", "static", "const", "struct", "sizeof", "unsigned", "long", "double", "float"];
    keywords.forEach(keyword => {
      const regex = new RegExp(`\\b(${keyword})\\b`, "g");
      html = html.replace(regex, '<span class="text-cyan-400 font-semibold">$1</span>');
    });

    html = html.replace(/\b(wasm_main|MILF_EXPORT|__attribute__|visibility|used)\b/g, '<span class="text-purple-400 font-semibold">$1</span>');

    // Restore comments highlighted
    comments.forEach((comment, index) => {
      html = html.replace(`___COMMENT_PLACEHOLDER_${index}___`, `<span class="text-emerald-500/80 italic">${comment}</span>`);
    });

    return html;
  }, [code]);

  const lines = code.split("\n");

  return (
    <div className={cn("relative overflow-hidden transition-all border-b border-border/30 bg-black/40", expanded ? "" : "max-h-56")}>
      <div className="flex font-mono text-[11px] leading-relaxed select-text overflow-x-auto">
        {/* Line numbers */}
        <div className="py-4 text-right pr-3 pl-4 text-muted-foreground/35 border-r border-border/10 select-none shrink-0 min-w-[38px] bg-black/10">
          {lines.map((_, i) => (
            <div key={i}>{i + 1}</div>
          ))}
        </div>

        {/* Code column */}
        <pre className="py-4 pl-4 pr-6 text-foreground/90 whitespace-pre flex-1 overflow-x-auto">
          <code dangerouslySetInnerHTML={{ __html: highlightedCodeHtml }} />
        </pre>
      </div>

      {!expanded && (
        <div className="absolute bottom-0 inset-x-0 h-16 bg-gradient-to-t from-black/90 via-black/50 to-transparent flex items-end justify-center pb-2.5">
          <button
            onClick={() => setExpanded(true)}
            className="text-[10px] font-semibold text-primary-foreground bg-primary/20 hover:bg-primary/30 border border-primary/30 px-3 py-1 rounded-full backdrop-blur-sm transition-all shadow-sm active:scale-95"
          >
            Show full code ↓
          </button>
        </div>
      )}
    </div>
  );
};

function DemoTemplateCard({
  template,
  onUse,
}: {
  template: DemoTemplate;
  onUse: (t: DemoTemplate) => void;
}) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(template.code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const Icon = template.icon;

  return (
    <div className="bg-surface border border-border rounded-xl overflow-hidden flex flex-col transition-all hover:border-border/80 hover:shadow-lg hover:shadow-black/20 group">
      {/* Card header */}
      <div className="p-5 pb-4 border-b border-border/50">
        <div className="flex items-start justify-between mb-3">
          <div className={cn("w-10 h-10 rounded-lg border flex items-center justify-center shrink-0", template.iconBg)}>
            <Icon className={cn("h-5 w-5", template.iconColor)} />
          </div>
          <span className={cn("text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-full", DIFFICULTY_STYLES[template.difficulty])}>
            {template.difficulty}
          </span>
        </div>

        <h3 className="font-mono font-semibold text-base text-foreground">{template.name}</h3>
        <p className="text-xs text-primary mt-0.5 font-medium">{template.tagline}</p>
        <p className="text-xs text-muted-foreground mt-2 leading-relaxed">{template.description}</p>

        <div className="flex flex-wrap gap-1.5 mt-3">
          {template.badges.map((b) => (
            <span
              key={b}
              className="text-[10px] font-mono px-1.5 py-0.5 bg-muted/60 border border-border/60 rounded text-muted-foreground"
            >
              {b}
            </span>
          ))}
        </div>
      </div>

      {/* Code block */}
      <div className="flex-1">
        <div className="flex items-center justify-between px-4 py-2 bg-black/20 border-b border-border/30">
          <div className="flex items-center gap-1.5">
            <div className="w-2.5 h-2.5 rounded-full bg-red-500/70" />
            <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/70" />
            <div className="w-2.5 h-2.5 rounded-full bg-green-500/70" />
            <span className="text-[10px] font-mono text-muted-foreground ml-2">source.c</span>
          </div>
          <button
            onClick={handleCopy}
            className="flex items-center gap-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
          >
            {copied ? (
              <><Check className="h-3 w-3 text-green-400" /><span className="text-green-400">Copied</span></>
            ) : (
              <><Copy className="h-3 w-3" /><span>Copy</span></>
            )}
          </button>
        </div>

        <CCodeViewer code={template.code} expanded={expanded} setExpanded={setExpanded} />

        {expanded && (
          <div className="flex justify-center py-2 bg-black/10">
            <button
              onClick={() => setExpanded(false)}
              className="text-[10px] font-semibold text-primary/70 hover:text-primary transition-colors bg-primary/10 border border-primary/20 px-3 py-1 rounded-full active:scale-95"
            >
              Collapse Code ↑
            </button>
          </div>
        )}
      </div>

      {/* Input / Output panels */}
      <div className="grid grid-cols-2 divide-x divide-border/40 border-t border-border/40">
        <div className="p-3">
          <p className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">Sample Input</p>
          <pre className="text-[10px] font-mono text-foreground/70 leading-relaxed whitespace-pre-wrap break-all">{template.sampleInput}</pre>
        </div>
        <div className="p-3">
          <p className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">Expected Output</p>
          <pre className="text-[10px] font-mono text-emerald-400/80 leading-relaxed whitespace-pre-wrap break-all">{template.expectedOutput}</pre>
        </div>
      </div>

      {/* ABI explanation note */}
      {template.abiExplanation && (
        <div className="px-5 py-3 border-t border-border/40 bg-blue-500/5 text-[11px] text-blue-400 leading-relaxed">
          <p className="font-semibold mb-1 flex items-center gap-1.5 text-blue-300">
            <Info className="h-3.5 w-3.5" />
            Why does wasm_main have 4 arguments?
          </p>
          <div className="whitespace-pre-line text-muted-foreground font-sans">
            {template.abiExplanation}
          </div>
        </div>
      )}

      {/* CTA */}
      <div className="px-5 py-3 border-t border-border/40 bg-muted/10">
        <Button
          size="sm"
          className="w-full gap-2 text-xs font-semibold"
          onClick={() => onUse(template)}
        >
          <Rocket className="h-3.5 w-3.5" />
          Use This Template
        </Button>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// DemoFunctionsTab
// ─────────────────────────────────────────────────────────────────────────────

function DemoFunctionsTab({ onUseTemplate }: { onUseTemplate: (t: DemoTemplate) => void }) {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-gradient-to-br from-primary/5 via-transparent to-purple-500/5 border border-primary/10 rounded-xl p-6">
        <div className="flex items-start gap-4">
          <div className="w-12 h-12 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0">
            <BookOpen className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-foreground">Demo Functions</h2>
            <p className="text-sm text-muted-foreground mt-1 leading-relaxed max-w-2xl">
              Explore five ready-to-run WASM templates — from a simple integer adder to a full JPEG-to-PDF renderer.
              Each template shows the complete source code, sample input, and expected output. Click{" "}
              <strong className="text-foreground">Use This Template</strong> to deploy one instantly.
            </p>
          </div>
        </div>

        {/* Progression path */}
        <div className="flex items-center gap-2 mt-5 flex-wrap">
          {["simple-add", "echo-json", "fetch-text", "url-to-pdf", "image-to-pdf", "multi-images-to-pdf"].map((name, i, arr) => (
            <div key={name} className="flex items-center gap-2">
              <span className="text-xs font-mono text-muted-foreground bg-muted/60 border border-border/60 px-2 py-1 rounded">
                {name}
              </span>
              {i < arr.length - 1 && <ChevronRight className="h-3 w-3 text-muted-foreground/40 shrink-0" />}
            </div>
          ))}
        </div>
      </div>

      {/* Cards grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
        {DEMO_TEMPLATES.map((t) => (
          <DemoTemplateCard key={t.id} template={t} onUse={onUseTemplate} />
        ))}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// FirstFunctionGuide — shown when My Functions is empty
// ─────────────────────────────────────────────────────────────────────────────

function FirstFunctionGuide({ onBrowseDemo }: { onBrowseDemo: () => void }) {
  const steps = [
    {
      n: "1",
      icon: Code2,
      title: "Pick a template or write from scratch",
      desc: "Choose one of the demo templates below, or open the editor and write your C/WASM function using milf.h.",
    },
    {
      n: "2",
      icon: Rocket,
      title: "Deploy to the mobile node fleet",
      desc: "Click Deploy — your code compiles to WASM in seconds and is pushed to connected Android edge nodes.",
    },
    {
      n: "3",
      icon: Terminal,
      title: "Invoke and see live output",
      desc: "Hit Invoke with a payload and watch the execution result come back in real time from the edge.",
    },
  ];

  return (
    <div className="py-6 space-y-8">
      {/* Hero */}
      <div className="text-center space-y-4 pb-2">
        <div className="relative inline-flex">
          <div className="w-20 h-20 rounded-2xl bg-primary/10 border border-primary/20 flex items-center justify-center">
            <Sparkles className="h-9 w-9 text-primary" />
          </div>
          <span className="absolute -top-1 -right-1 w-5 h-5 bg-green-500 border-2 border-background rounded-full animate-pulse" />
        </div>
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-foreground">Write Your First WASM Function</h2>
          <p className="text-sm text-muted-foreground mt-2 max-w-md mx-auto leading-relaxed">
            Pocket Cloud runs your code on real Android devices in{" "}
            <span className="text-primary font-semibold">&lt; 5ms</span> — no VMs, no cold-start overhead. Let's
            deploy your first serverless function.
          </p>
        </div>
      </div>

      {/* Steps */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 max-w-3xl mx-auto">
        {steps.map(({ n, icon: Icon, title, desc }) => (
          <div
            key={n}
            className="relative bg-surface border border-border rounded-xl p-5 text-center hover:border-primary/30 transition-colors"
          >
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground text-sm font-bold flex items-center justify-center mx-auto mb-3">
              {n}
            </div>
            <div className="w-10 h-10 rounded-xl bg-muted/40 flex items-center justify-center mx-auto mb-3">
              <Icon className="h-5 w-5 text-primary" />
            </div>
            <h3 className="text-sm font-semibold text-foreground mb-1">{title}</h3>
            <p className="text-xs text-muted-foreground leading-relaxed">{desc}</p>
          </div>
        ))}
      </div>

      {/* CTA row */}
      <div className="flex items-center justify-center gap-3 flex-wrap">
        <Button asChild size="lg" className="gap-2 font-semibold shadow-lg shadow-primary/20">
          <Link to="/functions/create">
            <Plus className="h-4 w-4" />
            Create from Scratch
          </Link>
        </Button>
        <Button variant="outline" size="lg" className="gap-2" onClick={onBrowseDemo}>
          <BookOpen className="h-4 w-4" />
          Browse Demo Functions
          <ArrowRight className="h-3.5 w-3.5" />
        </Button>
      </div>

      {/* Feature pills */}
      <div className="flex items-center justify-center gap-3 flex-wrap pt-2">
        {[
          { icon: Zap, text: "< 5ms cold start" },
          { icon: Box, text: "WASM sandboxed" },
          { icon: Network, text: "Mobile edge nodes" },
        ].map(({ icon: Icon, text }) => (
          <div key={text} className="flex items-center gap-1.5 text-xs text-muted-foreground bg-muted/30 border border-border/50 px-3 py-1.5 rounded-full">
            <Icon className="h-3 w-3" />
            {text}
          </div>
        ))}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Main Functions Page
// ─────────────────────────────────────────────────────────────────────────────

type ActiveTab = "my-functions" | "demo" | "docs";

// ─── Post-invocation notice banner ──────────────────────────────────────────

function PostInvokeNotice({
  fn,
  onDismiss,
}: {
  fn: FunctionEntity;
  onDismiss: () => void;
}) {
  // Auto-dismiss after 15 seconds
  useEffect(() => {
    const t = setTimeout(onDismiss, 15_000);
    return () => clearTimeout(t);
  }, [onDismiss]);

  return (
    <div
      className={cn(
        "fixed bottom-6 right-6 z-50",
        "flex items-start gap-4",
        "bg-background border border-primary/30 rounded-xl shadow-2xl shadow-primary/10",
        "p-4 w-[360px] max-w-[calc(100vw-2rem)]",
        "animate-in slide-in-from-bottom-4 fade-in duration-300",
      )}
    >
      {/* Pulsing status dot */}
      <div className="mt-0.5 shrink-0 relative">
        <div className="w-3 h-3 rounded-full bg-blue-500" />
        <div className="absolute inset-0 w-3 h-3 rounded-full bg-blue-500 animate-ping opacity-60" />
      </div>

      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-foreground leading-snug">
          Task dispatched —&nbsp;
          <span className="font-mono text-primary">{fn.name}</span>
        </p>
        <p className="text-xs text-muted-foreground mt-1 leading-relaxed">
          Waiting for a mobile node to pick it up. Open the{" "}
          <strong className="text-foreground">Invoke / Test</strong> tab to watch the live output.
        </p>
        <div className="flex items-center gap-2 mt-3">
          <Link
            to={`/functions/${fn.id}?tab=invoke`}
            className="flex items-center gap-1.5 text-xs font-semibold text-primary-foreground bg-primary hover:bg-primary/90 px-3 py-1.5 rounded-lg transition-colors"
          >
            <Play className="h-3 w-3" />
            View Live Output
          </Link>
          <button
            onClick={onDismiss}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors px-2 py-1.5"
          >
            Dismiss
          </button>
        </div>
      </div>

      {/* Close X */}
      <button
        onClick={onDismiss}
        className="shrink-0 text-muted-foreground hover:text-foreground transition-colors mt-0.5"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

export default function Functions() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedInvoke, setSelectedInvoke] = useState<FunctionEntity | null>(null);
  const [searchParams, setSearchParams] = useSearchParams();
  const rawTab = searchParams.get("tab");
  const activeTab = (rawTab === "demo" || rawTab === "docs" ? rawTab : "my-functions") as ActiveTab;
  const [lastInvokedFn, setLastInvokedFn] = useState<FunctionEntity | null>(null);

  const handleTabChange = (tab: ActiveTab) => {
    setSearchParams({ tab });
  };

  const { data: functions = [], isLoading, error } = useFunctions(searchQuery);
  const deleteFunction = useDeleteFunction();
  const invokeFunction = useInvokeFunction();

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this function?")) {
      deleteFunction.mutate(id);
    }
  };

  const handleInvokeSubmit = async (fn: FunctionEntity, payload: string) => {
    try {
      const result = await invokeFunction.mutateAsync({ id: fn.id, input: payload });
      // Show small toast
      toast({ title: "Dispatched!", description: `Execution ID: ${result.execution_id ?? "queued"}` });
      // Show the persistent notice banner pointing to the Invoke/Test tab
      setLastInvokedFn(fn);
    } catch (e) {
      toast({ title: "Invocation failed", description: (e as Error).message, variant: "destructive" });
    }
  };

  const handleUseTemplate = (template: DemoTemplate) => {
    // Store template code in sessionStorage so CreateFunction can pick it up
    sessionStorage.setItem("milf-template-code", template.code);
    sessionStorage.setItem("milf-template-runtime", template.runtime);
    sessionStorage.setItem("milf-template-name", template.name);
    navigate("/functions/create?from=template");
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case "active": return "Active";
      case "error": return "Error";
      case "inactive": return "Inactive";
      default: return status;
    }
  };

  const columns = [
    {
      key: "name",
      header: "Name",
      render: (fn: FunctionEntity) => (
        <Link
          to={`/functions/${fn.id}`}
          className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-muted/40 hover:bg-primary/10 border border-border/50 hover:border-primary/30 transition-all duration-200 group/tile active:scale-[0.97] hover:shadow-sm"
        >
          <Code2 className="h-3.5 w-3.5 text-muted-foreground group-hover/tile:text-primary transition-colors shrink-0" />
          <span className="font-mono text-sm font-semibold tracking-tight text-foreground group-hover/tile:text-primary transition-colors">
            {fn.name}
          </span>
          <ArrowRight className="h-3 w-3 text-primary opacity-0 -translate-x-1 group-hover/tile:opacity-100 group-hover/tile:translate-x-0 transition-all duration-250 shrink-0" />
        </Link>
      ),
    },
    {
      key: "runtime",
      header: "Runtime",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground font-mono text-xs">{fn.runtime}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      render: (fn: FunctionEntity) => (
        <StatusBadge status={fn.status === "active" ? "success" : fn.status === "error" ? "error" : "inactive"}>
          {getStatusLabel(fn.status)}
        </StatusBadge>
      ),
    },
    {
      key: "invocations",
      header: "Invocations",
      className: "text-right",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground">{fn.invocations24h.toLocaleString()}</span>
      ),
    },
    {
      key: "avgDuration",
      header: "Avg Duration",
      className: "text-right",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground">
          {fn.avgDurationMs ? `${Math.round(fn.avgDurationMs)}ms` : "-"}
        </span>
      ),
    },
    {
      key: "lastRun",
      header: "Last Run",
      className: "text-right",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground">
          {fn.lastRunAt ? formatDistanceToNow(new Date(fn.lastRunAt), { addSuffix: true }) : "-"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      className: "w-10",
      render: (fn: FunctionEntity) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-7 w-7">
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => navigate(`/functions/${fn.id}`)}>
              <ExternalLink className="h-3.5 w-3.5 mr-2" />
              View Details
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setSelectedInvoke(fn)}>
              <Play className="h-3.5 w-3.5 mr-2" />
              Invoke
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Copy className="h-3.5 w-3.5 mr-2" />
              Duplicate
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive" onClick={() => handleDelete(fn.id)}>
              <Trash2 className="h-3.5 w-3.5 mr-2" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  return (
    <AppLayout>
      {/* Guided tour — auto-starts on first visit */}
      <GuidedTour />

      {/* Post-invocation notice — slides up from bottom-right after invoking from this page */}
      {lastInvokedFn && (
        <PostInvokeNotice
          fn={lastInvokedFn}
          onDismiss={() => setLastInvokedFn(null)}
        />
      )}

      <PageHeader
        title="Functions"
        description="Manage your serverless functions"
        actions={
          <div className="flex items-center gap-3">
            <TourRestartButton className="hidden sm:flex" />
            <Button asChild id="create-function-btn">
              <Link to="/functions/create">
                <Plus className="h-4 w-4 mr-2" />
                Create Function
              </Link>
            </Button>
          </div>
        }
      />

      {/* Tab bar */}
      <div className="flex items-center gap-1 border-b border-border mb-6">
        {/* My Functions */}
        <button
          id="tab-my-functions"
          onClick={() => handleTabChange("my-functions")}
          className={cn(
            "flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors",
            activeTab === "my-functions"
              ? "border-primary text-foreground"
              : "border-transparent text-muted-foreground hover:text-foreground hover:border-border"
          )}
        >
          <Zap className="h-3.5 w-3.5" />
          My Functions
          {!isLoading && (
            <span className={cn(
              "text-[10px] font-bold px-1.5 py-0.5 rounded-full min-w-[18px] text-center",
              functions.length > 0 ? "bg-primary/15 text-primary" : "bg-muted/60 text-muted-foreground"
            )}>
              {functions.length}
            </span>
          )}
        </button>

        {/* Demo Functions */}
        <button
          id="tab-demo"
          onClick={() => handleTabChange("demo")}
          className={cn(
            "flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors",
            activeTab === "demo"
              ? "border-primary text-foreground"
              : "border-transparent text-muted-foreground hover:text-foreground hover:border-border"
          )}
        >
          <BookOpen className="h-3.5 w-3.5" />
          Demo Functions
          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded-full bg-primary/15 text-primary">
            {DEMO_TEMPLATES.length}
          </span>
        </button>

        {/* Docs */}
        <button
          id="tab-docs"
          onClick={() => handleTabChange("docs")}
          className={cn(
            "flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors",
            activeTab === "docs"
              ? "border-primary text-foreground"
              : "border-transparent text-muted-foreground hover:text-foreground hover:border-border"
          )}
        >
          <GraduationCap className="h-3.5 w-3.5" />
          Docs
          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/20">
            Guide
          </span>
        </button>
      </div>

      {/* My Functions Tab */}
      {activeTab === "my-functions" && (
        <>
          {isLoading ? (
            <div className="p-8 flex items-center justify-center text-muted-foreground">
              <Loader2 className="h-6 w-6 animate-spin mr-2" />
              Loading functions...
            </div>
          ) : functions.length === 0 && searchQuery === "" ? (
            <FirstFunctionGuide onBrowseDemo={() => handleTabChange("demo")} />
          ) : (
            <>
              {/* Search bar — only when there are functions */}
              <div className="flex items-center gap-4 mb-4">
                <div className="relative flex-1 max-w-sm">
                  <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                  <Input
                    type="search"
                    placeholder="Search functions..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-8 h-8 bg-background"
                  />
                </div>
              </div>

              <div className="bg-surface border border-border rounded-md">
                <DataTable
                  columns={columns}
                  data={functions}
                  emptyMessage="No functions match your search"
                />
              </div>
            </>
          )}
        </>
      )}

      {/* Demo Functions Tab */}
      {activeTab === "demo" && (
        <DemoFunctionsTab onUseTemplate={handleUseTemplate} />
      )}

      {/* Docs Tab */}
      {activeTab === "docs" && (
        <DocsTab />
      )}

      {/* Anchor used by guided tour step 5 (Invoke tab hint) */}
      <div id="invoke-tab-hint" className="sr-only" aria-hidden />

      {selectedInvoke && (
        <InvokeModal
          open={!!selectedInvoke}
          onClose={() => setSelectedInvoke(null)}
          functionName={selectedInvoke.name}
          runtime={selectedInvoke.runtime}
          sourceCode={selectedInvoke.source.type === "inline" ? selectedInvoke.source.code : ""}
          onInvoke={(payload) => handleInvokeSubmit(selectedInvoke, payload)}
          isLoading={invokeFunction.isPending}
        />
      )}
    </AppLayout>
  );
}
