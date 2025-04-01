# Benchmark Results

The python implementation of [wayback-discover-diff](https://github.com/internetarchive/wayback-discover-diff) was edited to log these statistics in a file.

| Language | URL        | Year | Total Captures | Processed Captures | Download Time (s) | Feature Extraction Time (s) | SimHash Calculation Time (s) | CDX Fetch Time (s) | Total Time (s) |
|----------|-----------|------|----------------|---------------------|-------------------|----------------------------|------------------------------|---------------------|---------------|
| Python   | github.com | 2023 | 1095           | 913                 | 8028.28           | 14.08                      | 1.95                         | 8.66                | 1034.31       |
| Python   | hello.com  | 2023 | 115            | 13                  | 14.67             | 0.0077                     | 0.0052                       | 12.51               | 15.04         |
| Python   | hello.com  | 2024 | 133            | 26                  | 40.80             | 0.0383                     | 0.0282                       | 4.39                | 8.71          |
