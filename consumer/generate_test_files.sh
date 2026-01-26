#!/bin/bash
mkdir -p test_assets/c_batch
for i in {1..100}
do
cat << EOC > test_assets/c_batch/job_$i.c
#include <stdio.h>
#include <stdlib.h>

int main() {
    printf("[JOB $i] Starting performance test...\n");
    int sum = 0;
    for(int j = 0; j < $i * 100; j++) {
        sum += j;
    }
    printf("Result of calculation: %d\n", sum);
    printf("Job $i completed successfully.\n");
    return 0;
}
EOC
done
echo "Generated 100 test files in test_assets/c_batch/"
