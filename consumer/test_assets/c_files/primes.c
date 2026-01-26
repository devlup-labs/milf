#include <stdio.h>

int is_prime(int n) {
  if (n <= 1)
    return 0;
  for (int i = 2; i * i <= n; i++) {
    if (n % i == 0)
      return 0;
  }
  return 1;
}

int main() {
  printf("Primes up to 20:\n");
  for (int i = 2; i <= 20; i++) {
    if (is_prime(i)) {
      printf("%d ", i);
    }
  }
  printf("\n");
  return 0;
}
