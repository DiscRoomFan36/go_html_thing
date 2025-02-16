package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func dump_string(s string, filename string) error {
	f, err := os.Create(filename)
	Assert(err == nil, err)
	defer f.Close()
	_, err = f.WriteString(s)
	return err
}

func get_html_from_url(url string) string {
	resp, err := http.Get(url)
	Assert(err == nil, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	Assert(err == nil, err)

	return string(body)
}

func main() {
	// url := "https://www.royalroad.com/fiction/72359/cartaflore/chapter/2059865/chapter-174-honest-red-reflection"
	// html_string := get_html_from_url(url)

	body, err := os.ReadFile("./whole_html.html")
	Assert(err == nil, err)

	html_doc, err := parse_HTML_Document(string(body))
	Assert(err == nil, err)

	s := fmt.Sprintf("%+v\n", html_doc)
	err = dump_string(s, "./new_thing.txt")
	Assert(err == nil, err)

	// const CHAPTER_INNER_CLASS = "<div class=\"chapter-inner chapter-content\">"

	// fmt.Printf("%+v\n", html_doc)

	print("Its all Good!\n")
}
