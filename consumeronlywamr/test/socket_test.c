
#include <stddef.h>
#include <stdint.h>

// =============================================================
// WASI Shim: Types and Import Definitions
// =============================================================

typedef int32_t __wasi_fd_t;
typedef int32_t __wasi_errno_t;

// Address families
#define AF_INET 2

// Socket types
#define SOCK_STREAM 1

typedef struct {
  uint8_t n0;
  uint8_t n1;
  uint8_t n2;
  uint8_t n3;
} __wasi_addr_ip4_t;

typedef struct {
  __wasi_addr_ip4_t addr;
  uint16_t port;
} __wasi_addr_ip4_port_t;

typedef struct {
  uint8_t kind; // 0 = IPv4
  __wasi_addr_ip4_port_t ip4;
} __wasi_addr_t;

typedef struct {
  uint8_t *buf;
  size_t buf_len;
} __wasi_ciovec_t;

// Standard BSD structures (simplified for IPv4)
struct in_addr {
  uint32_t s_addr;
};

struct sockaddr {
  uint16_t sa_family;
  char sa_data[14];
};

struct sockaddr_in {
  uint16_t sin_family;
  uint16_t sin_port;
  struct in_addr sin_addr;
  char sin_zero[8];
};

// WASI Syscalls Imports
#define WASI_IMPORT(name)                                                      \
  __attribute__((__import_module__("wasi_snapshot_preview1"),                  \
                 __import_name__(name)))

int32_t __wasi_sock_open(int32_t poolfd, int32_t af, int32_t socktype,
                         int32_t *sockfd) WASI_IMPORT("sock_open");
int32_t __wasi_sock_connect(int32_t fd, const __wasi_addr_t *addr)
    WASI_IMPORT("sock_connect");
int32_t __wasi_sock_send(int32_t fd, const __wasi_ciovec_t *si_data,
                         int32_t si_data_len, int32_t si_flags,
                         int32_t *so_data_len) WASI_IMPORT("sock_send");
int32_t __wasi_sock_recv(int32_t fd, const __wasi_ciovec_t *ri_data,
                         int32_t ri_data_len, int32_t ri_flags,
                         int32_t *ro_data_len, int32_t *ro_flags)
    WASI_IMPORT("sock_recv");
int32_t __wasi_fd_close(int32_t fd) WASI_IMPORT("fd_close");

// =============================================================
// Libc-like Wrappers
// =============================================================

int socket(int domain, int type, int protocol) {
  if (domain != AF_INET || type != SOCK_STREAM)
    return -1;

  int32_t sockfd;
  int ret = __wasi_sock_open(0, 0, 1, &sockfd);
  if (ret != 0)
    return -1;
  return sockfd;
}

uint16_t htons(uint16_t hostshort) {
  return (hostshort << 8) | (hostshort >> 8);
}

int connect(int sockfd, const struct sockaddr *addr, int addrlen) {
  const struct sockaddr_in *sin = (const struct sockaddr_in *)addr;

  __wasi_addr_t wasi_addr;
  wasi_addr.kind = 0; // IPv4

  // Extract bytes from s_addr (which is big endian/network order)
  uint32_t ip = sin->sin_addr.s_addr;
  uint8_t *p = (uint8_t *)&ip;
  wasi_addr.ip4.addr.n0 = p[0];
  wasi_addr.ip4.addr.n1 = p[1];
  wasi_addr.ip4.addr.n2 = p[2];
  wasi_addr.ip4.addr.n3 = p[3];

  wasi_addr.ip4.port = htons(sin->sin_port);

  return __wasi_sock_connect(sockfd, &wasi_addr);
}

int send(int sockfd, const void *buf, size_t len, int flags) {
  __wasi_ciovec_t vector;
  vector.buf = (uint8_t *)buf;
  vector.buf_len = len;

  int32_t sent;
  int ret = __wasi_sock_send(sockfd, &vector, 1, 0, &sent);
  if (ret != 0)
    return -1;
  return sent;
}

int recv(int sockfd, void *buf, size_t len, int flags) {
  __wasi_ciovec_t vector;
  vector.buf = (uint8_t *)buf;
  vector.buf_len = len;

  int32_t recvd;
  int32_t ro_flags;
  int ret = __wasi_sock_recv(sockfd, &vector, 1, 0, &recvd, &ro_flags);
  if (ret != 0)
    return -1;
  return recvd;
}

int close(int fd) { return __wasi_fd_close(fd); }

// Minimal stdlib
void *memcpy(void *dest, const void *src, size_t n) {
  char *d = (char *)dest;
  const char *s = (const char *)src;
  while (n--)
    *d++ = *s++;
  return dest;
}

void *memset(void *s, int c, size_t n) {
  unsigned char *p = (unsigned char *)s;
  while (n--)
    *p++ = (unsigned char)c;
  return s;
}

size_t strlen(const char *s) {
  size_t len = 0;
  while (*s++)
    len++;
  return len;
}

typedef struct {
  __wasi_addr_t addr;
  int32_t type;
} __wasi_addr_info_t;

int32_t __wasi_sock_addr_resolve(int32_t host, int32_t service, int32_t hints,
                                 int32_t addr_info, int32_t addr_info_size,
                                 int32_t *max_info_size)
    WASI_IMPORT("sock_addr_resolve");

// Helper to resolve simple
int resolve_google(struct in_addr *addr) {
  char *host = "google.com";
  __wasi_addr_info_t res[1];
  int32_t count = 0;

  int ret =
      __wasi_sock_addr_resolve((int32_t)host, 0, 0, (int32_t)res, 1, &count);

  if (ret != 0 || count == 0)
    return -1;

  if (res[0].addr.kind == 0) {
    uint8_t *p = (uint8_t *)&addr->s_addr;
    p[0] = res[0].addr.ip4.addr.n0;
    p[1] = res[0].addr.ip4.addr.n1;
    p[2] = res[0].addr.ip4.addr.n2;
    p[3] = res[0].addr.ip4.addr.n3;
    return 0;
  }
  return -1;
}

int32_t __wasi_fd_write(int32_t fd, const __wasi_ciovec_t *iovs,
                        int32_t iovs_len, int32_t *nwritten)
    WASI_IMPORT("fd_write");

void print(const char *str) {
  __wasi_ciovec_t iov;
  iov.buf = (uint8_t *)str;
  iov.buf_len = strlen(str);
  int32_t nw;
  __wasi_fd_write(1, &iov, 1, &nw);
}

// Export app_main for WAMR entry using int return to indicate success/fail
__attribute__((visibility("default"))) __attribute__((used)) int app_main() {
  print("Starting socket test (shimmed)...\n");

  int sock = socket(AF_INET, SOCK_STREAM, 0);
  if (sock == -1) {
    print("socket failed\n");
    return -1;
  }
  print("Socket created.\n");

  struct sockaddr_in server_addr;
  memset(&server_addr, 0, sizeof(server_addr));
  server_addr.sin_family = AF_INET;
  server_addr.sin_port = htons(80); // Port 80

  if (resolve_google(&server_addr.sin_addr) != 0) {
    server_addr.sin_addr.s_addr = 0x2eecf18e; // 142.241.236.46
    print("Resolve failed, using hardcoded IP.\n");
  } else {
    print("Resolved google.com.\n");
  }

  if (connect(sock, (struct sockaddr *)&server_addr, sizeof(server_addr)) < 0) {
    print("connect failed\n");
    close(sock);
    return -1;
  }
  print("Connected to google.com:80.\n");

  char *message =
      "GET / HTTP/1.1\r\nHost: google.com\r\nConnection: close\r\n\r\n";
  if (send(sock, message, strlen(message), 0) < 0) {
    print("send failed\n");
    close(sock);
    return -1;
  }
  print("Sent HTTP GET request.\n");

  char buffer[1024];
  int received = recv(sock, buffer, sizeof(buffer) - 1, 0);
  if (received < 0) {
    print("recv failed\n");
  } else {
    buffer[received] = '\0';
    print("Received bytes:\n");
    if (received > 100)
      buffer[100] = 0;
    print(buffer);
    print("\n");
  }

  close(sock);
  return 0;
}
