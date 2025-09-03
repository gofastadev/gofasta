#!/bin/bash

echo "Testing coverage extraction..."

# Run tests and generate coverage
cd /Users/descholar/descholar/myprojects/healtronlabs/gofasta
go test -v -coverprofile="test_coverage.out" ./tools/transpiler/core 2>/dev/null

if [[ -f "test_coverage.out" ]]; then
    echo "Coverage file created successfully"
    
    echo -e "\n=== Raw coverage output for parser.go ==="
    go tool cover -func="test_coverage.out" 2>/dev/null | grep "parser.go" | head -5
    
    echo -e "\n=== Attempting to extract file coverage ==="
    
    for target_file in parser.go ast_cache.go token_pool.go type_checker.go formatter.go import_cache.go; do
        echo -n "Checking $target_file: "
        
        # Get the last line for this file (should be the one with highest line number)
        last_line=$(go tool cover -func="test_coverage.out" 2>/dev/null | grep "${target_file}" | tail -1)
        
        if [[ -n "$last_line" ]]; then
            # Extract the percentage
            coverage=$(echo "$last_line" | awk '{print $NF}')
            echo "$coverage"
        else
            echo "NOT FOUND"
        fi
    done
    
    echo -e "\n=== Alternative: Using average approach ==="
    
    for target_file in parser.go ast_cache.go token_pool.go type_checker.go formatter.go import_cache.go; do
        echo -n "$target_file: "
        
        # Get all percentages for this file
        percentages=$(go tool cover -func="test_coverage.out" 2>/dev/null | grep "${target_file}" | awk '{print $NF}' | grep '%' | sed 's/%//')
        
        if [[ -n "$percentages" ]]; then
            # Calculate average
            sum=0
            count=0
            for pct in $percentages; do
                sum=$(echo "$sum + $pct" | bc 2>/dev/null || echo "$sum")
                count=$((count + 1))
            done
            
            if [[ $count -gt 0 ]]; then
                avg=$(echo "scale=1; $sum / $count" | bc 2>/dev/null || echo "0.0")
                echo "${avg}%"
            else
                echo "No percentages found"
            fi
        else
            echo "NOT FOUND"
        fi
    done
    
    # Clean up
    rm -f test_coverage.out
else
    echo "Failed to create coverage file"
fi