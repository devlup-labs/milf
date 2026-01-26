#include <stdio.h>
#include <stdlib.h>

int main() {
    printf("[JOB 73] Starting performance test...\n");
    int sum = 0;
    for(int j = 0; j < 73 * 100; j++) {
        sum += j;
    }
    printf("Result of calculation: %d\n", sum);
    printf("Job 73 completed successfully.\n");
    return 0;
}
