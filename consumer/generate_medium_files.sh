#!/bin/bash
mkdir -p test_assets/c_batch
for i in {1..3}
do
cat << EOC > test_assets/c_batch/medium_$i.c
#include <stdio.h>

int main() {
    printf("STRESS TEST: Requesting 200MB Medium Allocation Slot $i...\n");
    return 0;
}
EOC
done
echo "Generated 3 medium test files."
