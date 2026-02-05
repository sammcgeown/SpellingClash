#!/bin/bash

# SpellingClash Test Runner
# This script runs all tests in the project with coverage reporting

set -e

echo "🧪 Running SpellingClash Tests..."
echo "=================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Run tests with coverage
echo -e "${YELLOW}Running unit tests...${NC}"
go test ./internal/... -v -coverprofile=coverage.out -covermode=atomic

# Check if tests passed
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi

# Generate coverage report
echo -e "\n${YELLOW}Generating coverage report...${NC}"
go tool cover -html=coverage.out -o coverage.html

# Display coverage summary
echo -e "\n${YELLOW}Coverage Summary:${NC}"
go tool cover -func=coverage.out | tail -n 1

echo -e "\n${GREEN}Coverage report generated: coverage.html${NC}"
echo -e "${YELLOW}Open coverage.html in your browser to view detailed coverage${NC}"

# Optional: Check for minimum coverage threshold
COVERAGE=$(go tool cover -func=coverage.out | tail -n 1 | awk '{print $3}' | sed 's/%//')
THRESHOLD=50

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo -e "${RED}Warning: Coverage ($COVERAGE%) is below threshold ($THRESHOLD%)${NC}"
else
    echo -e "${GREEN}Coverage ($COVERAGE%) meets threshold ($THRESHOLD%)${NC}"
fi

echo -e "\n${GREEN}Tests complete!${NC}"
