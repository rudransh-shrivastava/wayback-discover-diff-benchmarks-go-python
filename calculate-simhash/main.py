import os
import time
import base64
import hashlib
import json
from collections import defaultdict
from datetime import datetime
from itertools import groupby
from selectolax.parser import HTMLParser
from simhash import Simhash

def custom_hash_function(x):
    """Custom hash function using BLAKE2b."""
    return int.from_bytes(hashlib.blake2b(x).digest(), byteorder='big')

def extract_html_features(html_content):
    """Extract features from HTML content."""
    tree = HTMLParser(html_content)
    tree.strip_tags(['script', 'style'])
    text = tree.root.text(separator=' ')
    if not text:
        return {}
    text = text.lower()
    text = text.translate(str.maketrans('', '', string.punctuation))
    lines = (line.strip() for line in text.splitlines())
    chunks = (phrase.strip() for line in lines for phrase in line.split("  "))
    text = '\n'.join(chunk for chunk in chunks if chunk)
    words = sorted(text.split())
    features = {}
    for k, g in groupby(words):
        features[k] = sum(1 for _ in g)
    return features

def calculate_simhash(features, simhash_size):
    """Calculate SimHash from features using custom hash function."""
    return Simhash(features, simhash_size, hashfunc=custom_hash_function).value

def process_html_file(file_path, simhash_size=64):
    """Process a single HTML file and return timing metrics."""
    result = {
        'file_read_time': 0.0,
        'feature_extraction_time': 0.0,
        'simhash_calculation_time': 0.0,
        'simhash_encoding_time': 0.0,
        'total_processing_time': 0.0,
        'feature_count': 0,
        'simhash': '',
        'error': None
    }
    try:
        # Read file
        start_time = time.time()
        with open(file_path, 'rb') as f:
            html_content = f.read().decode('utf-8', errors='ignore')
        result['file_read_time'] = time.time() - start_time

        # Extract features
        start_time = time.time()
        features = extract_html_features(html_content)
        result['feature_extraction_time'] = time.time() - start_time
        result['feature_count'] = len(features)

        if not features:
            result['error'] = "No features extracted"
            return result

        # Calculate SimHash
        start_time = time.time()
        simhash_value = calculate_simhash(features, simhash_size)
        result['simhash_calculation_time'] = time.time() - start_time

        # Encode SimHash to base64
        start_time = time.time()
        simhash_bytes = simhash_value.to_bytes(8, byteorder='little')
        simhash_b64 = base64.b64encode(simhash_bytes).decode('utf-8')
        result['simhash_encoding_time'] = time.time() - start_time

        result['simhash'] = simhash_b64
        result['total_processing_time'] = (
            result['file_read_time'] +
            result['feature_extraction_time'] +
            result['simhash_calculation_time'] +
            result['simhash_encoding_time']
        )
    except Exception as e:
        result['error'] = str(e)
    return result

def benchmark_html_processing(folder_path, simhash_size=64, max_files=5):
    """Benchmark HTML processing for files in the specified folder."""
    results = {}
    summary = {
        'total_benchmark_time': 0.0,
        'files_processed': 0,
        'average_file_processing_time': 0.0,
    }
    total_start_time = time.time()

    try:
        files = [f for f in os.listdir(folder_path) if f.endswith('.html')][:max_files]
    except Exception as e:
        results['error'] = {'error': f"Failed to list directory: {str(e)}"}
        return results, summary

    total_processing_time = 0.0
    processed_files = 0

    for file_name in files:
        file_path = os.path.join(folder_path, file_name)
        file_result = process_html_file(file_path, simhash_size)
        results[file_name] = file_result
        if file_result.get('error') is None:
            processed_files += 1
            total_processing_time += file_result['total_processing_time']

    summary['total_benchmark_time'] = time.time() - total_start_time
    summary['files_processed'] = processed_files
    summary['average_file_processing_time'] = total_processing_time / processed_files if processed_files > 0 else 0.0

    return results, summary

def compress_captures(captures):
    """Compress timestamp and SimHash pairs."""
    hash_dict = {}
    grouped = defaultdict(lambda: defaultdict(lambda: defaultdict(list)))

    for ts, simhash in captures:
        year = int(ts[:4])
        month = int(ts[4:6])
        day = int(ts[6:8])
        hms = ts[8:]
        hash_id = hash_dict.get(simhash, len(hash_dict))
        if simhash not in hash_dict:
            hash_dict[simhash] = hash_id
        grouped[year][month][day].append([hms, hash_id])

    new_captures = []
    for year in sorted(grouped.keys()):
        year_entry = [year]
        for month in sorted(grouped[year].keys()):
            month_entry = [month]
            for day in sorted(grouped[year][month].keys()):
                day_entry = [day] + grouped[year][month][day]
                month_entry.append(day_entry)
            year_entry.append(month_entry)
        new_captures.append(year_entry)

    sorted_hashes = sorted(hash_dict.items(), key=lambda x: x[1])
    hashes = [item[0] for item in sorted_hashes]

    return {
        'captures': new_captures,
        'hashes': hashes
    }

def main():
    print("Starting HTML SimHash benchmark...")
    folder_path = 'pages'
    simhash_size = 64
    max_files = 5  # Process up to 5 files for benchmarking

    results, summary = benchmark_html_processing(folder_path, simhash_size, max_files)

    if 'error' in results:
        print(f"Error: {results['error']['error']}")
        return

    print("\n=== HTML SimHash Benchmark Results ===")
    print(f"Total files processed: {summary['files_processed']}")
    print(f"Total benchmark time: {summary['total_benchmark_time']:.4f} seconds")
    print(f"Average file processing time: {summary['average_file_processing_time']:.4f} seconds")

    print("\nDetailed per-file results:")
    for file_name, result in results.items():
        if file_name == 'error':
            continue
        print(f"\n--- {file_name} ---")
        if result.get('error'):
            print(f"Error: {result['error']}")
            continue
        print(f"File read time: {result['file_read_time']:.4f} sec")
        print(f"Feature extraction time: {result['feature_extraction_time']:.4f} sec")
        print(f"SimHash calculation time: {result['simhash_calculation_time']:.4f} sec")
        print(f"SimHash encoding time: {result['simhash_encoding_time']:.4f} sec")
        print(f"Total processing time: {result['total_processing_time']:.4f} sec")
        print(f"Feature count: {result['feature_count']}")
        print(f"SimHash: {result['simhash']}")

    # Compression demo
    captures = []
    for file_name, result in results.items():
        if file_name == 'error' or result.get('error'):
            continue
        now = datetime.now().strftime("%Y%m%d%H%M%S")
        captures.append((now, result['simhash']))

    if captures:
        compressed = compress_captures(captures)
        print("\n=== Compressed Captures Demo ===")
        print(f"Original captures count: {len(captures)}")
        print(f"Unique hashes count: {len(compressed['hashes'])}")
        if compressed['hashes']:
            print(f"First few hashes: {compressed['hashes'][:3]}")

if __name__ == "__main__":
    import string
    main()
