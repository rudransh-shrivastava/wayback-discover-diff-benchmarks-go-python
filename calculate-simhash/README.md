# Benchmark Results

The python module was copied over from the original [wayback-discover-diff](https://github.com/internetarchive/wayback-discover-diff) repository.
The Golang code was poorly written by AI just for the sake of comparison with the Python implementation.

| Benchmark Run | Total Files Processed | Total Benchmark Time (s) | Average File Processing Time (s) |
|--------------|----------------------|-------------------------|--------------------------------|
| Golang       | 2                    | 0.0057                  | 0.0028                         |
| Python       | 2                    | 0.0348                  | 0.0173                         |

Golang was `6.1 times` faster than Python with average file processing time also being `6.1 times` faster.

# Raw Golang Results

```bash
Starting HTML SimHash benchmark...

=== HTML SimHash Benchmark Results ===
Total files processed: 2
Total benchmark time: 0.0057 seconds
Average file processing time: 0.0028 seconds

Detailed per-file results:

--- SimHash - Wikipedia.html ---
File read time: 0.0001 seconds
Feature extraction time: 0.0021 seconds
SimHash calculation time: 0.0001 seconds
SimHash encoding time: 0.0000 seconds
Total processing time: 0.0023 seconds
Feature count: 401
SimHash: Pjbr9h5Tcog=

--- Wikipedia_What is an article_ - Wikipedia.html ---
File read time: 0.0003 seconds
Feature extraction time: 0.0028 seconds
SimHash calculation time: 0.0002 seconds
SimHash encoding time: 0.0000 seconds
Total processing time: 0.0033 seconds
Feature count: 789
SimHash: PjbrqB7SOog=

=== Compressed Captures Demo ===
Original captures count: 2
Unique hashes count: 2
First few hashes: [PjbrqB7SOog= Pjbr9h5Tcog=]
```

# Raw Python Results
```bash
=== HTML SimHash Benchmark Results ===
Total files processed: 2
Total benchmark time: 0.0348 seconds
Average file processing time: 0.0173 seconds

Detailed per-file results:

--- Wikipedia_What is an article_ - Wikipedia.html ---
File read time: 0.0019 sec
Feature extraction time: 0.0198 sec
SimHash calculation time: 0.0032 sec
SimHash encoding time: 0.0000 sec
Total processing time: 0.0249 sec
Feature count: 858
SimHash: J1jXIKnsyL4=

--- SimHash - Wikipedia.html ---
File read time: 0.0007 sec
Feature extraction time: 0.0075 sec
SimHash calculation time: 0.0015 sec
SimHash encoding time: 0.0000 sec
Total processing time: 0.0096 sec
Feature count: 397
SimHash: Fwm7KKKfzRY=

=== Compressed Captures Demo ===
Original captures count: 2
Unique hashes count: 2
First few hashes: ['J1jXIKnsyL4=', 'Fwm7KKKfzRY=']
```
