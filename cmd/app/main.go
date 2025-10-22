package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	crawler "example.com/m/cmd/internal/ftpcrawler"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Templates struct {
	templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() *Templates {
	return &Templates{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
}

type FormData struct {
	Values, Errors map[string]string
}

func newFormData() FormData {
	return FormData{
		Values: make(map[string]string),
		Errors: make(map[string]string),
	}
}

type file struct {
	Path  string   // path to the file
	Terms []string // list of terms (converted to [][]string)
}

type Results struct {
	Results    []file
	TotalFiles int
	TotalTerms int
	Error      string
	JobID      string
	Duration   float64
}

type ProgressUpdate struct {
	ScannedFiles int
	MatchedFiles int
	MatchedFile  *MatchedFileData
}

type MatchedFileData struct {
	Path  string   `json:"path"`
	Terms []string `json:"terms"`
}

type Job struct {
	ProgressChan chan ProgressUpdate
	Results      *Results
	Done         bool
	Error        error
	StartTime    time.Time
	mu           sync.Mutex
}

// type Page struct {
// 	Data Results
// 	Form FormData
// }

// func convertByteSlicesToStrings(byteSlices [][]byte) []string {
// 	stringSlices := make([]string, len(byteSlices))
// 	for i, byteSlice := range byteSlices {
// 		stringSlices[i] = string(byteSlice)
// 	}
// 	return stringSlices
// }

func countAndFormatTerms(byteSlices [][]byte) ([]string, int) {
	termCount := make(map[string]int)
	totalTerms := 0
	for _, byteSlice := range byteSlices {
		term := string(byteSlice)
		termCount[term]++
	}

	var formattedTerms []string
	for term, count := range termCount {
		formattedTerms = append(formattedTerms, fmt.Sprintf("%s - %d", term, count))
		totalTerms += count
	}

	// fmt.Println("Total Terms: ", totalTerms)
	return formattedTerms, totalTerms
}

// Define the CustomField struct
type CustomField struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// Define the Page struct
type Page struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	CustomFields []CustomField `json:"customFields"`
	DueDate      string        `json:"dueDate"`
}

// Create a function to parse the incoming JSON data and populate the Page struct
func newPageFromData(data []byte) (*Page, error) {
	var pageData Page
	if err := json.Unmarshal(data, &pageData); err != nil {
		return nil, err
	}
	return &pageData, nil
}
func main() {
	// Jobs map to track in-progress crawls
	var jobsMu sync.Mutex
	jobs := make(map[string]*Job)

	// Cleanup goroutine to remove old jobs after 30 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			jobsMu.Lock()
			for id, job := range jobs {
				job.mu.Lock()
				if job.Done && time.Since(time.Now()) > 30*time.Minute {
					close(job.ProgressChan)
					delete(jobs, id)
				}
				job.mu.Unlock()
			}
			jobsMu.Unlock()
		}
	}()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Renderer = newTemplate()

	e.Static("/assets", "assets")
	e.Static("/css", "css")

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})

	e.POST("/ftp", func(c echo.Context) error {
		host := c.FormValue("host")
		user := c.FormValue("user")
		password := c.FormValue("password")
		path := c.FormValue("path")
		terms := c.FormValue("terms")

		// trim whitespace
		hostT := strings.TrimSpace(host)
		userT := strings.TrimSpace(user)
		passwordT := strings.TrimSpace(password)
		pathT := strings.TrimSpace(path)
		termsT := strings.TrimSpace(terms)

		// Generate unique job ID
		jobID := uuid.New().String()

		// Create job with progress channel
		job := &Job{
			ProgressChan: make(chan ProgressUpdate, 100),
			Results:      nil,
			Done:         false,
			Error:        nil,
			StartTime:    time.Now(),
		}

		// Store job
		jobsMu.Lock()
		jobs[jobID] = job
		jobsMu.Unlock()

		// Launch FTP crawl in goroutine
		go func() {
			// Create a wrapper channel to convert crawler.ProgressUpdate to main.ProgressUpdate
			crawlerProgressChan := make(chan crawler.ProgressUpdate, 100)

			// Launch goroutine to convert progress updates
			go func() {
				for update := range crawlerProgressChan {
					var matchedFile *MatchedFileData = nil
					if update.MatchedFile != nil {
						formattedTerms, _ := countAndFormatTerms(update.MatchedFile.Terms)
						matchedFile = &MatchedFileData{
							Path:  update.MatchedFile.Path,
							Terms: formattedTerms,
						}
					}

					mainUpdate := ProgressUpdate{
						ScannedFiles: update.ScannedFiles,
						MatchedFiles: update.MatchedFiles,
						MatchedFile:  matchedFile,
					}

					select {
					case job.ProgressChan <- mainUpdate:
					default:
					}
				}
			}()

			rawResults, err := crawler.FtpCrawl(hostT, userT, passwordT, pathT, termsT, crawlerProgressChan)
			close(crawlerProgressChan)

			job.mu.Lock()
			defer job.mu.Unlock()

			duration := time.Since(job.StartTime).Seconds()

			if err != nil {
				job.Error = err
				job.Results = &Results{
					Results:    []file{},
					TotalFiles: 0,
					TotalTerms: 0,
					Error:      err.Error(),
					JobID:      jobID,
					Duration:   duration,
				}
				job.Done = true
				return
			}

			// Process results
			results := Results{
				Results:    make([]file, len(rawResults)),
				TotalFiles: len(rawResults),
				TotalTerms: 0,
				Error:      "",
				JobID:      jobID,
				Duration:   duration,
			}

			total := 0
			for i, rawResult := range rawResults {
				formattedTerms, totalTerms := countAndFormatTerms(rawResult.Terms)
				total += totalTerms
				results.Results[i] = file{
					Path:  rawResult.Path,
					Terms: formattedTerms,
				}
			}
			results.TotalTerms = total

			job.Results = &results
			job.Done = true
		}()

		// Return initial loading state with job ID
		initialResults := Results{
			JobID: jobID,
		}
		return c.Render(http.StatusOK, "progress-template", initialResults)
	})

	// SSE endpoint for progress updates
	e.GET("/progress/:jobId", func(c echo.Context) error {
		jobID := c.Param("jobId")
		fmt.Println("SSE connection opened for job:", jobID)

		// Get job
		jobsMu.Lock()
		job, exists := jobs[jobID]
		jobsMu.Unlock()

		if !exists {
			fmt.Println("Job not found:", jobID)
			return c.String(http.StatusNotFound, "Job not found")
		}

		// Set SSE headers
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().Header().Set("X-Accel-Buffering", "no")
		c.Response().WriteHeader(http.StatusOK)

		// Send initial connection event
		fmt.Fprintf(c.Response(), "event: connected\ndata: {\"message\": \"connected\"}\n\n")
		c.Response().Flush()

		// Stream progress updates
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		lastScanned := 0
		lastMatched := 0
		for {
			select {
			case progress, ok := <-job.ProgressChan:
				if ok {
					shouldSendProgress := progress.ScannedFiles > lastScanned || progress.MatchedFiles > lastMatched

					if shouldSendProgress {
						lastScanned = progress.ScannedFiles
						lastMatched = progress.MatchedFiles
						fmt.Printf("Progress update for job %s: %d scanned, %d matched\n", jobID, progress.ScannedFiles, progress.MatchedFiles)
						fmt.Fprintf(c.Response(), "event: progress\ndata: {\"scannedFiles\": %d, \"matchedFiles\": %d}\n\n", progress.ScannedFiles, progress.MatchedFiles)
						c.Response().Flush()
					}

					// Always send match data if a file was matched (independent of progress counter)
					if progress.MatchedFile != nil {
						matchData, err := json.Marshal(progress.MatchedFile)
						if err == nil {
							fmt.Printf("Sending match-found event: %s\n", progress.MatchedFile.Path)
							fmt.Fprintf(c.Response(), "event: match-found\ndata: %s\n\n", string(matchData))
							c.Response().Flush()
						} else {
							fmt.Printf("Error marshaling match data: %v\n", err)
						}
					}
				}
			case <-ticker.C:
				job.mu.Lock()
				done := job.Done
				results := job.Results
				job.mu.Unlock()

				if done {
					fmt.Println("Job complete:", jobID)
					// Send final results
					if results != nil {
						if results.Error != "" {
							// Send error
							fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", results.Error)
						} else {
							// Render results template
							var buf strings.Builder
							err := e.Renderer.(*Templates).templates.ExecuteTemplate(&buf, "results", results)
							if err == nil {
								// Escape newlines for SSE
								htmlData := strings.ReplaceAll(buf.String(), "\n", " ")
								htmlData = strings.ReplaceAll(htmlData, "\r", "")
								fmt.Fprintf(c.Response(), "event: complete\ndata: %s\n\n", htmlData)
							} else {
								fmt.Println("Error rendering template:", err)
							}
						}
						c.Response().Flush()
					}
					return nil
				}

				// Send heartbeat with current progress
				if lastScanned > 0 {
					fmt.Fprintf(c.Response(), "event: progress\ndata: {\"scannedFiles\": %d, \"matchedFiles\": %d}\n\n", lastScanned, lastMatched)
					c.Response().Flush()
				}
			}
		}
	})

	e.Logger.Fatal(e.Start(":42069"))
}
