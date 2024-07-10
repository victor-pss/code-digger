package crawler

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/jlaffaye/ftp"
)

// file struct is used to store the file path and the terms to search for
type file struct {
	Path  string   // path to the file
	Terms [][]byte // list of terms
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

func FtpCrawl(host, user, password, path, terms string) []file {
	var found = []file{}
	c, err := ftp.Dial(host + ":21")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Quit()

	err = c.Login(user, password)
	if err != nil {
		log.Fatal(err)
	}

	w := c.Walk(path)

	for w.Next() {
		if w.Err() != nil {
			log.Fatal(w.Err())
		}
		if w.Stat().Type.String() == "folder" {
			entries, err := c.List(w.Path())
			if err != nil {
				log.Fatal(err)
			}

			for _, entry := range entries {
				// if strings.HasSuffix(entry.Name, ".php") {
				if strings.HasSuffix(entry.Name, ".php") {
					fmt.Println(w.Path() + "/" + entry.Name)
					r, err := c.Retr(w.Path() + "/" + entry.Name)
					if err != nil {
						log.Fatal(err)
					}

					buf, err := io.ReadAll(r)
					if err != nil {
						log.Fatal(err)
					}

					// regex to find terms in each file
					// (?i) = case insensitive
					re, err := buildRegexFromTerms(terms)
					if err != nil {
						log.Fatal(err)
					}
					if re.FindAll(buf, -1) != nil {
						found = append(found, file{
							Path:  w.Path() + "/" + entry.Name,
							Terms: re.FindAll(buf, -1),
						})
					}

					err = r.Close()
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
	if err := c.Quit(); err != nil {
		log.Fatal(err)
	}
	return found
}
