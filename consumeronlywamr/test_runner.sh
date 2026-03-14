#!/bin/bash
# Quick Test Script - Test WASM Runner with Enhanced Metrics

set -e

echo "🧪 WASM Runner Testing Script"
echo "=============================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROJECT_ROOT="/Users/adarsh/Projects/personal/consumeronlywamr"
APK_PATH="$PROJECT_ROOT/android/app/build/outputs/apk/debug/app-debug.apk"
TEST_DIR="$PROJECT_ROOT/test"

# Step 1: Check ADB
echo -e "${BLUE}[1/5]${NC} Checking ADB connection..."
if ! adb devices | grep -q "device$"; then
    echo -e "${YELLOW}⚠️  No device connected. Please connect your Android device.${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Device connected"
echo ""

# Step 2: Install APK  
echo -e "${BLUE}[2/5]${NC} Installing latest APK with monitoring..."
if [ ! -f "$APK_PATH" ]; then
    echo -e "${YELLOW}⚠️  APK not found. Building...${NC}"
    cd "$PROJECT_ROOT/android"
    ./gradlew assembleDebug
fi

adb install -r "$APK_PATH"
echo -e "${GREEN}✓${NC} APK installed"
echo ""

# Step 3: Push test files
echo -e "${BLUE}[3/5]${NC} Pushing test WASM files..."
cd "$TEST_DIR"

# Find all .wasm files
WASM_FILES=$(find . -maxdepth 1 -name "*.wasm" -type f)

if [ -z "$WASM_FILES" ]; then
    echo -e "${YELLOW}⚠️  No WASM files found in $TEST_DIR${NC}"
else
    for file in $WASM_FILES; do
        adb push "$file" /sdcard/Download/ 2>/dev/null
        basename_file=$(basename "$file")
        echo -e "  ${GREEN}✓${NC} Pushed $basename_file"
    done
fi
echo ""

# Step 4: Clear logs
echo -e "${BLUE}[4/5]${NC} Clearing old logs..."
adb logcat -c
echo -e "${GREEN}✓${NC} Logs cleared"
echo ""

# Step 5: Start monitoring
echo -e "${BLUE}[5/5]${NC} Starting log monitor..."
echo ""
echo -e "${YELLOW}════════════════════════════════════════${NC}"
echo -e "${YELLOW}     Monitoring for execution metrics   ${NC}"
echo -e "${YELLOW}════════════════════════════════════════${NC}"
echo ""
echo "Now open the app and select a WASM file."
echo "Watch for execution metrics below:"
echo ""

# Monitor logs with colored output
adb logcat | grep --line-buffered -E "ExecutionMonitor|MemoryTracker|native-lib" | \
while IFS= read -r line; do
    if [[ $line == *"▶ Starting execution"* ]]; then
        echo -e "${GREEN}$line${NC}"
    elif [[ $line == *"■ Execution Summary"* ]]; then
        echo -e "${BLUE}$line${NC}"
    elif [[ $line == *"⚠️"* ]] || [[ $line == *"WARN"* ]]; then
        echo -e "${YELLOW}$line${NC}"
    elif [[ $line == *"ERROR"* ]] || [[ $line == *"🚨"* ]]; then
        echo -e "\033[0;31m$line${NC}"  # Red
    else
        echo "$line"
    fi
done
