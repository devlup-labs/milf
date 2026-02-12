#include <stdio.h>
#include <stdlib.h>

int main() {
    printf("[JOB 56] Starting performance test...\n");
    int sum = 0;
    for(int j = 0; j < 56 * 100; j++) {
        sum += j;
    }
    printf("Result of calculation: %d\n", sum);
    printf("Job 56 completed successfully.\n");
    return 0;
}
