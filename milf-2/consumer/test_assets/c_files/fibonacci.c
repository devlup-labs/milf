#include <stdio.h>

int main() {
  int n = 10, t1 = 0, t2 = 1, nextTerm;
  printf("Fibonacci Series: ");
  for (int i = 1; i <= n; ++i) {
    printf("%d, ", t1);
    nextTerm = t1 + t2;
    t1 = t2;
    t2 = nextTerm;
  }
  printf("\n");
  return 0;
}
