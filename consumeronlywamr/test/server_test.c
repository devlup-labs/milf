
#include <stddef.h>
#include <stdint.h>

// =============================================================
// WASI Shim & Imports (Same as client)
// =============================================================
typedef int32_t __wasi_fd_t;
typedef int32_t __wasi_errno_t;
#define AF_INET 2
#define SOCK_STREAM 1
#define WASI_IMPORT(name)                                                      \
  __attribute__((__import_module__("wasi_snapshot_preview1"),                  \
                 __import_name__(name)))

typedef struct {
  uint8_t n0, n1, n2, n3;
} __wasi_addr_ip4_t;
typedef struct {
  __wasi_addr_ip4_t addr;
  uint16_t port;
} __wasi_addr_ip4_port_t;
typedef struct {
  uint8_t kind;
  __wasi_addr_ip4_port_t ip4;
} __wasi_addr_t;
typedef struct {
  uint8_t *buf;
  size_t buf_len;
} __wasi_ciovec_t;

int32_t __wasi_sock_open(int32_t poolfd, int32_t af, int32_t socktype,
                         int32_t *sockfd) WASI_IMPORT("sock_open");
int32_t __wasi_sock_bind(int32_t fd, __wasi_addr_t *addr)
    WASI_IMPORT("sock_bind");
int32_t __wasi_sock_listen(int32_t fd, int32_t backlog)
    WASI_IMPORT("sock_listen");
int32_t __wasi_sock_accept(int32_t fd, int32_t flags, int32_t *fd_new)
    WASI_IMPORT("sock_accept");
int32_t __wasi_fd_write(int32_t fd, const __wasi_ciovec_t *iovs,
                        int32_t iovs_len, int32_t *nwritten)
    WASI_IMPORT("fd_write");

int socket(int domain, int type, int protocol) {
  int32_t sockfd;
  if (__wasi_sock_open(0, 0, 1, &sockfd) != 0)
    return -1;
  return sockfd;
}

uint16_t htons(uint16_t hostshort) {
  return (hostshort << 8) | (hostshort >> 8);
}

// Helper print
size_t strlen(const char *s) {
  size_t len = 0;
  while (*s++)
    len++;
  return len;
}
void print(const char *str) {
  __wasi_ciovec_t iov = {(uint8_t *)str, strlen(str)};
  int32_t nw;
  __wasi_fd_write(1, &iov, 1, &nw);
}

// Export main
__attribute__((visibility("default"))) __attribute__((used)) int main() {
  print("Starting server test...\n");

  int sockfd = socket(AF_INET, SOCK_STREAM, 0);
  if (sockfd == -1) {
    print("Socket creation failed\n");
    return 1;
  }

  // Bind to 0.0.0.0:12345
  __wasi_addr_t addr;
  addr.kind = 0; // IPv4
  addr.ip4.addr.n0 = 0;
  addr.ip4.addr.n1 = 0;
  addr.ip4.addr.n2 = 0;
  addr.ip4.addr.n3 = 0;
  addr.ip4.port = htons(12345);

  if (__wasi_sock_bind(sockfd, &addr) != 0) {
    print("Bind failed\n");
    return 1;
  }
  print("Bound to port 12345.\n");

  if (__wasi_sock_listen(sockfd, 3) != 0) {
    print("Listen failed\n");
    return 1;
  }

  // THIS IS THE "PORT READY" STATE
  print("Network port ready: Listening on 12345...\n");

  // Accept one connection for demo
  int new_fd;
  if (__wasi_sock_accept(sockfd, 0, &new_fd) == 0) {
    print("Accepted a connection!\n");
    // Handle connection...
  } else {
    print("Accept failed\n");
  }

  return 0;
}
