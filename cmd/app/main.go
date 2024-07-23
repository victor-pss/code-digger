package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
	// "github.com/labstack/echo/v4/middleware"
	// add ftpcrawl.go from ../internal/ftpcrawl.go
	crawler "example.com/m/cmd/internal/ftpcrawler"
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
	Results []file
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

func countAndFormatTerms(byteSlices [][]byte) []string {
	termCount := make(map[string]int)
	for _, byteSlice := range byteSlices {
		term := string(byteSlice)
		termCount[term]++
	}

	var formattedTerms []string
	for term, count := range termCount {
		formattedTerms = append(formattedTerms, fmt.Sprintf("%s - %d", term, count))
	}

	return formattedTerms
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

	e := echo.New()
	// e.Use(middleware.Logger())

	// page := newPage()
	e.Renderer = newTemplate()

	e.Static("/assets", "assets")
	e.Static("/css", "css")

	e.GET("/", func(c echo.Context) error {
		// return c.Render(200, "index.html", page)
		return c.Render(200, "index.html", nil)
	})

	e.POST("/ftp", func(c echo.Context) error {
		host := c.FormValue("host")
		user := c.FormValue("user")
		password := c.FormValue("password")
		path := c.FormValue("path")
		terms := c.FormValue("terms")

		rawResults := crawler.FtpCrawl(host, user, password, path, terms)

		// fmt.Print("pre")
		results := Results{
			Results: make([]file, len(rawResults)),
		}

		for i, rawResult := range rawResults {
			formattedTerms := countAndFormatTerms(rawResult.Terms) // Count and format terms
			results.Results[i] = file{
				Path:  rawResult.Path, // Access the correct Path field
				Terms: formattedTerms, // Store the formatted terms
			}
		}

		fmt.Println(results)
		return c.Render(200, "results", results)
	})
	e.Logger.Fatal(e.Start(":42069"))
}
