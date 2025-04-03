# Device Info
**NOTE:** All benchmarks were conducted on a machine with the following specifications:

- CPU: 12th Gen Intel i3-1215U (8) @ 4.400GHz
- RAM: 16GB
- OS: GNU/Linux

# Ratelimit
**NOTE:** Benchmarks including capture download and CDX fetching were rate-limited by the Wayback Machine's API

# Key findings:

- Golang performed 6x faster than Python in SimHash calculation: [Info](https://github.com/rudransh-shrivastava/wayback-discover-diff-benchmarks-go-python/tree/main/calculate-simhash)
- Significant time was required for fetching captures and CDX: [Info](https://github.com/rudransh-shrivastava/wayback-discover-diff-benchmarks-go-python/tree/main/fetch-captures)
