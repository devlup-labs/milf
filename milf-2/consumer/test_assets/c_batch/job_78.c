#include <stdio.h>
#include <stdlib.h>

int main() {
    printf("[JOB 78] Starting performance test...\n");
    int sum = 0;
    for(int j = 0; j < 78 * 100; j++) {
        sum += j;
    }
    printf("Result of calculation: %d\n", sum);
    printf("Job 78 completed successfully.\n");
    return 0;
}
