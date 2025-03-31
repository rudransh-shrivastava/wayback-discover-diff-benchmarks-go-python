package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config represents the application configuration
type Config struct {
	Concurrency      int
	MaxRetries       int
	Timeout          time.Duration
	MaxCaptureSize   int64
	BenchmarkDir     string
	SnapshotsPerYear int
}

// Capture represents a wayback machine capture
type Capture struct {
	Timestamp string
	Digest    string
}

// CaptureResult stores the result of a capture download
type CaptureResult struct {
	Timestamp    string  `json:"timestamp"`
	Digest       string  `json:"digest,omitempty"`
	DownloadTime float64 `json:"download_time"`
	Size         int     `json:"size,omitempty"`
	ContentType  string  `json:"content_type,omitempty"`
	Error        string  `json:"error,omitempty"`
	StatusCode   int     `json:"status_code,omitempty"`
}

// BenchmarkResult stores the complete benchmark data
type BenchmarkResult struct {
	URL             string          `json:"url"`
	Year            string          `json:"year"`
	Timestamp       string          `json:"timestamp"`
	Summary         SummaryMetrics  `json:"summary"`
	DetailedTimings []CaptureResult `json:"detailed_capture_timings"`
}

// SummaryMetrics contains the summary performance metrics
type SummaryMetrics struct {
	Captures     CaptureMetrics `json:"captures"`
	CDXFetchTime float64        `json:"cdx_fetch_time"`
	TotalTime    float64        `json:"total_time"`
}

// CaptureMetrics tracks overall capture processing metrics
type CaptureMetrics struct {
	Total             int     `json:"total"`
	Processed         int     `json:"processed"`
	DownloadTime      float64 `json:"download_time"`
	SuccessfulFetches int     `json:"successful_fetches"`
	FailedFetches     int     `json:"failed_fetches"`
}

// Client wraps the HTTP client with wayback-specific operations
type Client struct {
	httpClient *http.Client
	config     Config
	userAgent  string
}

func NewClient(config Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        config.Concurrency,
		MaxIdleConnsPerHost: config.Concurrency,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &Client{
		httpClient: httpClient,
		config:     config,
		userAgent:  "wayback-discover-diff-go",
	}
}

// FetchCDX retrieves a list of captures for a URL in a specific year
func (c *Client) FetchCDX(url, year string) ([]Capture, error) {
	cdxURL := "https://web.archive.org/web/timemap"

	req, err := http.NewRequest("GET", cdxURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating CDX request: %v", err)
	}

	// Set query parameters
	q := req.URL.Query()
	q.Set("url", url)
	q.Set("from", year)
	q.Set("to", year)
	q.Set("statuscode", "200")
	q.Set("fl", "timestamp,digest")
	q.Set("collapse", "timestamp:9")

	if c.config.SnapshotsPerYear > 0 {
		q.Set("limit", fmt.Sprintf("%d", c.config.SnapshotsPerYear))
	}

	req.URL.RawQuery = q.Encode()

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept-Encoding", "gzip,deflate")
	req.Header.Set("Connection", "keep-alive")

	log.Printf("Fetching CDX for %s for year %s", url, year)
	startTime := time.Now()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CDX request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CDX request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CDX response: %v", err)
	}

	// Parse the CDX response
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	captures := make([]Capture, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			captures = append(captures, Capture{
				Timestamp: parts[0],
				Digest:    parts[1],
			})
		}
	}

	log.Printf("CDX fetch completed in %.2f seconds, found %d captures",
		time.Since(startTime).Seconds(), len(captures))

	return captures, nil
}

// DownloadCapture downloads a specific capture from the Wayback Machine with retries
func (c *Client) DownloadCapture(timestamp, url string) (CaptureResult, []byte, error) {
	captureURL := fmt.Sprintf("https://web.archive.org/web/%sid_/%s", timestamp, url)
	result := CaptureResult{Timestamp: timestamp}

	var data []byte
	var err error

	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		startTime := time.Now()

		req, reqErr := http.NewRequest("GET", captureURL, nil)
		if reqErr != nil {
			result.Error = fmt.Sprintf("error creating request: %v", reqErr)
			result.DownloadTime = time.Since(startTime).Seconds()
			return result, nil, reqErr
		}

		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept-Encoding", "gzip,deflate")
		req.Header.Set("Connection", "keep-alive")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("[Attempt %d] Request failed for %s: %v", attempt, timestamp, err)
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
			continue
		}
		defer resp.Body.Close()

		result.StatusCode = resp.StatusCode
		if resp.StatusCode != http.StatusOK {
			log.Printf("[Attempt %d] Unexpected status code %d for %s", attempt, resp.StatusCode, timestamp)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		// Read response body with a limit
		limitedReader := io.LimitReader(resp.Body, c.config.MaxCaptureSize)
		data, err = io.ReadAll(limitedReader)
		if err != nil {
			log.Printf("[Attempt %d] Failed to read response for %s: %v", attempt, timestamp, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		result.Size = len(data)
		result.DownloadTime = time.Since(startTime).Seconds()
		result.ContentType = resp.Header.Get("Content-Type")

		// Only return content for HTML responses
		if strings.Contains(strings.ToLower(result.ContentType), "text/html") ||
			strings.Contains(strings.ToLower(result.ContentType), "text") {
			return result, data, nil
		}

		// For non-HTML content, return metadata but no content
		return result, nil, nil
	}

	// If all retries fail, return last error
	result.Error = fmt.Sprintf("Failed after %d retries", c.config.MaxRetries)
	return result, nil, err
}

// processCapturesParallel processes captures in parallel using a worker pool
func (c *Client) processCapturesParallel(url string, captures []Capture) []CaptureResult {
	totalCaptures := len(captures)
	results := make([]CaptureResult, 0, totalCaptures)
	resultsChan := make(chan CaptureResult, totalCaptures)

	var wg sync.WaitGroup

	// Channel to distribute work
	jobs := make(chan Capture, totalCaptures)

	// Launch worker goroutines
	for w := 0; w < c.config.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for capture := range jobs {
				result, _, err := c.DownloadCapture(capture.Timestamp, url)
				if err != nil {
					result.Digest = capture.Digest
				} else {
					result.Digest = capture.Digest
				}
				resultsChan <- result
			}
		}()
	}

	// Send all jobs to the workers
	for _, capture := range captures {
		jobs <- capture
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

func main() {
	url := flag.String("url", "", "URL to fetch from Wayback Machine")
	year := flag.String("year", "", "Year to fetch captures for")
	concurrency := flag.Int("concurrency", 50, "Number of concurrent downloads")
	timeout := flag.Int("timeout", 20, "Timeout in seconds for HTTP requests")
	maxSize := flag.Int64("max-size", 1000000, "Maximum capture size to download")
	snapshotsPerYear := flag.Int("snapshots", -1, "Number of snapshots per year (-1 for all)")

	flag.Parse()

	if *url == "" || *year == "" {
		log.Fatal("URL and year are required parameters")
	}

	config := Config{
		Concurrency:      *concurrency,
		MaxRetries:       2,
		Timeout:          time.Duration(*timeout) * time.Second,
		MaxCaptureSize:   *maxSize,
		BenchmarkDir:     "benchmarks-go",
		SnapshotsPerYear: *snapshotsPerYear,
	}

	// Ensure benchmark directory exists
	os.MkdirAll(config.BenchmarkDir, 0755)

	client := NewClient(config)

	startTime := time.Now()

	benchmark := BenchmarkResult{
		URL:       *url,
		Year:      *year,
		Timestamp: time.Now().Format(time.RFC3339),
		Summary: SummaryMetrics{
			Captures: CaptureMetrics{},
		},
	}

	// Fetch CDX
	cdxStartTime := time.Now()
	captures, err := client.FetchCDX(*url, *year)
	if err != nil {
		log.Fatalf("Failed to fetch CDX: %v", err)
	}
	benchmark.Summary.CDXFetchTime = time.Since(cdxStartTime).Seconds()

	benchmark.Summary.Captures.Total = len(captures)

	results := client.processCapturesParallel(*url, captures)

	// Update metrics
	var totalDownloadTime float64
	successfulFetches := 0
	failedFetches := 0

	for _, result := range results {
		totalDownloadTime += result.DownloadTime

		if result.Error == "" {
			successfulFetches++
		} else {
			failedFetches++
		}
	}

	benchmark.Summary.Captures.DownloadTime = totalDownloadTime
	benchmark.Summary.Captures.SuccessfulFetches = successfulFetches
	benchmark.Summary.Captures.FailedFetches = failedFetches
	benchmark.Summary.Captures.Processed = successfulFetches
	benchmark.Summary.TotalTime = time.Since(startTime).Seconds()
	benchmark.DetailedTimings = results

	// Save benchmark results
	safeURL := strings.ReplaceAll(strings.ReplaceAll(*url, ":", "_"), "/", "_")
	benchmarkFile := filepath.Join(config.BenchmarkDir, fmt.Sprintf("%s_%s.json", safeURL, *year))

	f, err := os.Create(benchmarkFile)
	if err != nil {
		log.Fatalf("Failed to create benchmark file: %v", err)
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(benchmark); err != nil {
		log.Fatalf("Failed to write benchmark results: %v", err)
	}
	f.Close()

	log.Printf("Benchmark complete for %s, year %s", *url, *year)
	log.Printf("Total time: %.2f seconds", benchmark.Summary.TotalTime)
	log.Printf("CDX fetch time: %.2f seconds", benchmark.Summary.CDXFetchTime)
	log.Printf("Total captures: %d", benchmark.Summary.Captures.Total)
	log.Printf("Successful fetches: %d", benchmark.Summary.Captures.SuccessfulFetches)
	log.Printf("Failed fetches: %d", benchmark.Summary.Captures.FailedFetches)
	log.Printf("Total download time: %.2f seconds", benchmark.Summary.Captures.DownloadTime)
	log.Printf("Results saved to: %s", benchmarkFile)
}
