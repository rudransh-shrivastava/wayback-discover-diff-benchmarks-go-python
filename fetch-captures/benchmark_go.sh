#!/bin/bash

# List of websites and years to benchmark
declare -a urls=("youtube.com")
declare -a years=("2023")

echo "Starting Go Benchmark..." > benchmarks-go/logs/go_benchmark.log

start_time=$(date +%s)

for url in "${urls[@]}"; do
  for year in "${years[@]}"; do
    echo "Benchmarking Go: $url ($year)"

    go run main.go --url "$url" --year "$year" > /dev/null
  done
done

wait

end_time=$(date +%s)
duration=$((end_time - start_time))

echo "Total time: $duration sec" >> benchmarks-go/logs/go_benchmark.log

echo "Go Benchmark Complete!"
