#include <stdio.h>
#include <stdlib.h>

int main() {
    printf("[JOB 80] Starting performance test...\n");
    int sum = 0;
    for(int j = 0; j < 80 * 100; j++) {
        sum += j;
    }
    printf("Result of calculation: %d\n", sum);
    printf("Job 80 completed successfully.\n");
    return 0;
}
