#!/bin/bash

# List of websites and years to benchmark
declare -a urls=("youtube.com")
declare -a years=("2023")

echo "Starting Python Benchmark..." > benchmarks-python/logs/python_benchmark.log

for url in "${urls[@]}"; do
  for year in "${years[@]}"; do
    echo "Benchmarking Python: $url ($year)"
    start_time=$(date +%s)

        job_id=$(curl -s "https://wayback-api.archive.org/services/calculate-simhash?url=$url&year=$year" | jq -r '.job_id')

        while true; do
          status_info=$(curl -s "https://wayback-api.archive.org/services/job?job_id=$job_id")
          status=$(echo "$status_info" | jq -r '.status')
          info=$(echo "$status_info" | jq -r '.info')

          echo "Job $job_id status: $status - $info"

          if [[ "$status" == "SUCCESS" ]]; then
            break
          fi
          if [[ "$status" == "error" ]]; then
            echo "Error fetching: $info"
            break
          fi
          sleep 1  # Check every second
        done

    end_time=$(date +%s)
    duration=$((end_time - start_time))

    echo "$url, $year, $duration sec" >> benchmarks-python/logs/python_benchmark.log
  done
done

echo "Python Benchmark Complete!"
