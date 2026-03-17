#include <stdint.h>
#include <string.h>
#include <stdio.h>

// This is the new standard signature for MILF functions handling strings/JSON:
// - payload: The input string (e.g. "{\"name\":\"Adarsh\"}")
// - payload_len: Length of the input string
// - out_buf: A pre-allocated buffer where we write our response
// - out_max: The maximum capacity of our response buffer
__attribute__((visibility("default"))) __attribute__((used)) 
int wasm_main(char* payload, int payload_len, char* out_buf, int out_max) {
    // 1. Log something to stdout (appears in Android Logcat)
    printf("C Code: Received payload of length %d\n", payload_len);

    // 2. Wrap the payload in a new JSON response
    // payload_len is the size of the incoming string from the host.
    // We already null-terminated it in the C++ bridge.
    int written = snprintf(out_buf, out_max, 
        "{ \"status\": \"success\", \"message\": \"I am WASM\", \"original_data\": %s }", 
        payload);

    // 3. Return the exact length written
    return (written >= out_max) ? out_max - 1 : written;
}
