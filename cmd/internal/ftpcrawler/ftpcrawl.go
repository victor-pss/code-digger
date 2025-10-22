package crawler

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/jlaffaye/ftp"
)

// file struct is used to store the file path and the terms to search for
type file struct {
	Path  string   // path to the file
	Terms [][]byte // list of terms
}

type ProgressUpdate struct {
	ScannedFiles int
	MatchedFiles int
	MatchedFile  *file // nil if no match, populated if match found
}

func buildRegexFromTerms(terms string) (*regexp.Regexp, error) {
	// Split the terms by commas and trim spaces
	splitTerms := strings.Split(terms, ",")
	for i, term := range splitTerms {
		splitTerms[i] = strings.TrimSpace(term)
	}

	// Join the terms into a regex pattern
	pattern := "(?i)" + strings.Join(splitTerms, "|")
	return regexp.Compile(pattern)
}

func FtpCrawl(host, user, password, path, terms string, progressChan chan<- ProgressUpdate) ([]file, error) {
	var found = []file{}
	var filesScanned = 0
	var filesMatched = 0
	c, err := ftp.Dial(host + ":21")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP server: %w", err)
	}
	defer c.Quit()

	err = c.Login(user, password)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	w := c.Walk(path)

	for w.Next() {
		if w.Err() != nil {
			return found, fmt.Errorf("error walking directory: %w", w.Err())
		}
		if w.Stat().Type.String() == "folder" {
			entries, err := c.List(w.Path())
			if err != nil {
				return found, fmt.Errorf("error listing directory: %w", err)
			}

			for _, entry := range entries {
				// if strings.HasSuffix(entry.Name, ".php") {
				if strings.HasSuffix(entry.Name, ".php") || strings.HasSuffix(entry.Name, ".js") {
					fmt.Println(w.Path() + "/" + entry.Name)
					filesScanned++

					r, err := c.Retr(w.Path() + "/" + entry.Name)
					if err != nil {
						return found, fmt.Errorf("error retrieving file: %w", err)
					}

					buf, err := io.ReadAll(r)
					if err != nil {
						return found, fmt.Errorf("error reading file: %w", err)
					}

					// regex to find terms in each file
					// (?i) = case insensitive
					re, err := buildRegexFromTerms(terms)
					if err != nil {
						return found, fmt.Errorf("error building regex: %w", err)
					}

					matchedTerms := re.FindAll(buf, -1)
					var matchedFile *file = nil

					if matchedTerms != nil {
						filesMatched++
						newFile := file{
							Path:  w.Path() + "/" + entry.Name,
							Terms: matchedTerms,
						}
						found = append(found, newFile)
						matchedFile = &newFile
					}

					// Send progress update if channel is provided
					if progressChan != nil {
						select {
						case progressChan <- ProgressUpdate{
							ScannedFiles: filesScanned,
							MatchedFiles: filesMatched,
							MatchedFile:  matchedFile,
						}:
						default:
							// Channel is full or closed, continue without blocking
						}
					}

					err = r.Close()
					if err != nil {
						return found, fmt.Errorf("error closing file: %w", err)
					}
				}
			}
		}
	}
	if err := c.Quit(); err != nil {
		return found, fmt.Errorf("error closing FTP connection: %w", err)
	}
	return found, nil
}
