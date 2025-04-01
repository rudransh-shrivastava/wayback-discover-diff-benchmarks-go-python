package main

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/mfonda/simhash" // using mfonda/simhash package
	"golang.org/x/net/html"
)

// HTMLFeatures represents the word frequencies extracted from HTML.
type HTMLFeatures map[string]int

// BenchmarkResult represents the timing and results of processing one file.
type BenchmarkResult struct {
	FileReadTime           float64
	FeatureExtractionTime  float64
	SimHashCalculationTime float64
	SimHashEncodingTime    float64
	TotalProcessingTime    float64
	FeatureCount           int
	SimHash                string
	Error                  string
}

// BenchmarkSummary contains overall benchmark metrics.
type BenchmarkSummary struct {
	TotalBenchmarkTime        float64
	FilesProcessed            int
	AverageFileProcessingTime float64
}

// TimeCapture represents a timestamp and its corresponding SimHash.
type TimeCapture struct {
	Timestamp string
	SimHash   string
}

// CompressedCaptures is the compressed representation of the captures.
type CompressedCaptures struct {
	Captures []interface{}
	Hashes   []string
}

// simpleFeatureSet is a simple type that implements simhash.FeatureSet.
// It wraps a slice of simhash.Feature.
type simpleFeatureSet []simhash.Feature

// GetFeatures returns the underlying slice of features.
func (s simpleFeatureSet) GetFeatures() []simhash.Feature {
	return s
}

// extractHTMLFeatures processes HTML document and extracts key features as text.
func extractHTMLFeatures(htmlContent string) (HTMLFeatures, error) {
	features := make(HTMLFeatures)
	// Parse HTML.
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return features, err
	}

	// Create goquery document.
	gDoc := goquery.NewDocumentFromNode(doc)

	// Remove script and style tags.
	gDoc.Find("script, style").Remove()

	// Extract text.
	text := gDoc.Text()
	if text == "" {
		return features, nil
	}

	// Convert to lowercase.
	text = strings.ToLower(text)

	// Remove punctuation.
	text = strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, text)

	// Split into lines and process.
	lines := strings.Split(text, "\n")
	var processedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Split multi-headlines.
		chunks := strings.Split(line, "  ")
		for _, chunk := range chunks {
			chunk = strings.TrimSpace(chunk)
			if chunk != "" {
				processedLines = append(processedLines, chunk)
			}
		}
	}

	// Join and split into words.
	processedText := strings.Join(processedLines, "\n")
	words := strings.Fields(processedText)

	// Sort words and count frequencies.
	sort.Strings(words)
	currentWord := ""
	count := 0

	for _, word := range words {
		if word == currentWord {
			count++
		} else {
			if currentWord != "" {
				features[currentWord] = count
			}
			currentWord = word
			count = 1
		}
	}
	// Add the last word.
	if currentWord != "" {
		features[currentWord] = count
	}

	return features, nil
}

// calculateSimHash calculates SimHash for the given features.
// Note: The simHashSize parameter is now ignored because the library always
// produces a 64-bit hash.
func calculateSimHash(features HTMLFeatures, simHashSize int) uint64 {
	// Convert features to SimHash format using the provided helper.
	var featureList []simhash.Feature
	for word, weight := range features {
		// Use NewFeatureWithWeight to create a feature.
		featureList = append(featureList, simhash.NewFeatureWithWeight([]byte(word), weight))
	}

	// Wrap featureList in a simpleFeatureSet to satisfy the FeatureSet interface.
	fs := simpleFeatureSet(featureList)

	// Calculate and return the 64-bit SimHash.
	return simhash.Simhash(fs)
}

// hash calculates the hash of input data using SHA-512.
func hash(data []byte) uint64 {
	h := sha512.New()
	h.Write(data)
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

// packSimHashToBytes converts SimHash to bytes.
func packSimHashToBytes(simHash uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, simHash)
	return bytes
}

// processHTMLFile processes a single HTML file and returns timing metrics and SimHash.
func processHTMLFile(filePath string, simHashSize int) BenchmarkResult {
	result := BenchmarkResult{}

	// Step 1: Read the file.
	startTime := time.Now()
	htmlBytes, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to read file %s: %v", filePath, err)
		return result
	}
	htmlContent := string(htmlBytes)
	result.FileReadTime = time.Since(startTime).Seconds()

	// Step 2: Extract features.
	startTime = time.Now()
	features, err := extractHTMLFeatures(htmlContent)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to extract features from %s: %v", filePath, err)
		return result
	}
	result.FeatureExtractionTime = time.Since(startTime).Seconds()
	result.FeatureCount = len(features)

	// Step 3: Calculate SimHash.
	startTime = time.Now()
	simHashValue := calculateSimHash(features, simHashSize)
	result.SimHashCalculationTime = time.Since(startTime).Seconds()

	// Step 4: Pack SimHash to bytes and encode.
	startTime = time.Now()
	simHashBytes := packSimHashToBytes(simHashValue)
	result.SimHash = base64.StdEncoding.EncodeToString(simHashBytes)
	result.SimHashEncodingTime = time.Since(startTime).Seconds()

	result.TotalProcessingTime = result.FileReadTime + result.FeatureExtractionTime + result.SimHashCalculationTime + result.SimHashEncodingTime

	return result
}

// benchmarkHTMLProcessing benchmarks HTML processing for all files in a folder.
func benchmarkHTMLProcessing(folderPath string, simHashSize int) (map[string]BenchmarkResult, BenchmarkSummary) {
	results := make(map[string]BenchmarkResult)
	summary := BenchmarkSummary{}

	totalStartTime := time.Now()

	// Get list of files in the folder.
	files, err := os.ReadDir(folderPath)
	if err != nil {
		results["error"] = BenchmarkResult{
			Error: fmt.Sprintf("Failed to list directory %s: %v", folderPath, err),
		}
		return results, summary
	}

	// Process each file (up to 5).
	fileCount := 0
	totalProcessingTime := 0.0

	for _, file := range files {
		if file.IsDir() || fileCount >= 5 {
			continue
		}

		filePath := filepath.Join(folderPath, file.Name())
		startTime := time.Now()
		fileResult := processHTMLFile(filePath, simHashSize)
		fileResult.TotalProcessingTime = time.Since(startTime).Seconds()

		results[file.Name()] = fileResult
		totalProcessingTime += fileResult.TotalProcessingTime
		fileCount++
	}

	// Calculate overall metrics.
	summary.TotalBenchmarkTime = time.Since(totalStartTime).Seconds()
	summary.FilesProcessed = fileCount

	if fileCount > 0 {
		summary.AverageFileProcessingTime = totalProcessingTime / float64(fileCount)
	}

	return results, summary
}

// compressCaptures compresses timestamp and SimHash pairs.
func compressCaptures(captures []TimeCapture) CompressedCaptures {
	hashDict := make(map[string]int)
	grouped := make(map[int]map[int]map[int][]interface{})

	for _, capture := range captures {
		ts := capture.Timestamp
		simHash := capture.SimHash

		// Parse timestamp components.
		year, _ := strToInt(ts[0:4])
		month, _ := strToInt(ts[4:6])
		day, _ := strToInt(ts[6:8])
		hms := ts[8:]

		// Get or assign hash ID.
		hashID, exists := hashDict[simHash]
		if !exists {
			hashID = len(hashDict)
			hashDict[simHash] = hashID
		}

		// Create nested maps if they don't exist.
		if _, exists := grouped[year]; !exists {
			grouped[year] = make(map[int]map[int][]interface{})
		}
		if _, exists := grouped[year][month]; !exists {
			grouped[year][month] = make(map[int][]interface{})
		}
		if _, exists := grouped[year][month][day]; !exists {
			grouped[year][month][day] = make([]interface{}, 0)
		}

		// Add capture.
		cap := []interface{}{hms, hashID}
		grouped[year][month][day] = append(grouped[year][month][day], cap)
	}

	// Build compressed captures.
	compressedCaptures := CompressedCaptures{}

	// Sort years.
	var years []int
	for year := range grouped {
		years = append(years, year)
	}
	sort.Ints(years)

	// Build the nested structure.
	for _, year := range years {
		yearData := []interface{}{year}

		// Sort months.
		var months []int
		for month := range grouped[year] {
			months = append(months, month)
		}
		sort.Ints(months)

		for _, month := range months {
			monthData := []interface{}{month}

			// Sort days.
			var days []int
			for day := range grouped[year][month] {
				days = append(days, day)
			}
			sort.Ints(days)

			for _, day := range days {
				dayData := []interface{}{day}
				dayData = append(dayData, grouped[year][month][day]...)
				monthData = append(monthData, dayData)
			}

			yearData = append(yearData, monthData)
		}

		compressedCaptures.Captures = append(compressedCaptures.Captures, yearData)
	}

	// Sort hashes by hash ID.
	type hashPair struct {
		hash   string
		hashID int
	}
	var pairs []hashPair
	for hash, id := range hashDict {
		pairs = append(pairs, hashPair{hash, id})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].hashID < pairs[j].hashID
	})

	for _, pair := range pairs {
		compressedCaptures.Hashes = append(compressedCaptures.Hashes, pair.hash)
	}

	return compressedCaptures
}

// strToInt converts string to int safely.
func strToInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid digit: %c", c)
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

func main() {
	fmt.Println("Starting HTML SimHash benchmark...")

	// Run the benchmark.
	results, summary := benchmarkHTMLProcessing("pages/", 64)

	// Check for errors.
	if result, hasError := results["error"]; hasError {
		fmt.Printf("Error: %s\n", result.Error)
		return
	}

	// Print results.
	fmt.Println("\n=== HTML SimHash Benchmark Results ===")
	fmt.Printf("Total files processed: %d\n", summary.FilesProcessed)
	fmt.Printf("Total benchmark time: %.4f seconds\n", summary.TotalBenchmarkTime)
	fmt.Printf("Average file processing time: %.4f seconds\n", summary.AverageFileProcessingTime)

	fmt.Println("\nDetailed per-file results:")
	for fileName, result := range results {
		fmt.Printf("\n--- %s ---\n", fileName)
		if result.Error != "" {
			fmt.Printf("Error: %s\n", result.Error)
			continue
		}
		fmt.Printf("File read time: %.4f seconds\n", result.FileReadTime)
		fmt.Printf("Feature extraction time: %.4f seconds\n", result.FeatureExtractionTime)
		fmt.Printf("SimHash calculation time: %.4f seconds\n", result.SimHashCalculationTime)
		fmt.Printf("SimHash encoding time: %.4f seconds\n", result.SimHashEncodingTime)
		fmt.Printf("Total processing time: %.4f seconds\n", result.TotalProcessingTime)
		fmt.Printf("Feature count: %d\n", result.FeatureCount)
		fmt.Printf("SimHash: %s\n", result.SimHash)
	}

	// Create timestamp and simhash pairs for compression demo.
	var captures []TimeCapture
	for _, result := range results {
		if result.Error == "" {
			now := time.Now()
			timestamp := fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
				now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
			captures = append(captures, TimeCapture{
				Timestamp: timestamp,
				SimHash:   result.SimHash,
			})
		}
	}

	if len(captures) > 0 {
		fmt.Println("\n=== Compressed Captures Demo ===")
		compressedCaptures := compressCaptures(captures)
		fmt.Printf("Original captures count: %d\n", len(captures))
		fmt.Printf("Unique hashes count: %d\n", len(compressedCaptures.Hashes))
		if len(compressedCaptures.Hashes) > 0 {
			count := int(math.Min(3, float64(len(compressedCaptures.Hashes))))
			fmt.Printf("First few hashes: %v\n", compressedCaptures.Hashes[:count])
		}
	}
}
