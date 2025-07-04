#!/bin/bash

filter_coverage() {
    local input=${1:-coverage.out}
    local output=${2:-coverage.filtered.out}
    local ignore_file=".cover.ignore"

    if [[ ! -f "$input" ]]; then
        echo "❌ Coverage file not found: $input"
        return 1
    fi

    if [[ ! -f "$ignore_file" ]]; then
        echo "❌ Ignore file not found: $ignore_file"
        return 1
    fi

    # Read non-comment, non-empty lines and join as regex
    local pattern
    pattern=$(grep -vE '^\s*#|^\s*$' "$ignore_file" | paste -sd "|" -)

    if [[ -z "$pattern" ]]; then
        echo "⚠️  No ignore patterns found in $ignore_file"
        cp "$input" "$output"
        return 0
    fi

    # Filter: keep header, remove matching lines
    {
        head -n1 "$input"
        tail -n +2 "$input" | grep -Ev "$pattern"
    } > "$output"

    echo "✅ Filtered coverage written to: $output"
}

go test ./...
go test -coverprofile=coverage.out ./... 
filter_coverage coverage.out coverage.filtered.out
go tool cover -html=coverage.filtered.out -o coverage.html 
sed -i 's/black/whitesmoke/g' coverage.html
echo "✅ Coverage generated in coverage.html"