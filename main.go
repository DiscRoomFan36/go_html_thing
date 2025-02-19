package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
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

const HEADER_CLASS = "<h1 style=\"margin-top: 10px\" class=\"font-white break-word\">"
const CHAPTER_INNER_CLASS = "<div class=\"chapter-inner chapter-content\">"

func split_once(s string, sep string) (string, string, bool) {
	split := strings.SplitN(s, sep, 2)
	if len(split) == 1 {
		return s, "", false
	}
	return split[0], split[1], true
}

type Royal_Ident struct {
	fiction_ident string
	chapter_ident string
}

func parse_RoyalRoad_url_to_canonical(url string) Royal_Ident {
	Assert(url != "", "do not pass in the empty string")

	const URL_PREFIX = "https://www.royalroad.com/fiction/"
	url, ok := strings.CutPrefix(url, URL_PREFIX)
	Assert(ok, "url must start with prefix")

	fiction_ident, url, ok := split_once(url, "/")
	Assert(ok, "was not ok")

	for !strings.HasPrefix(url, "chapter/") {
		_, url, ok = split_once(url, "/")
		Assert(ok, "was not ok")
	}

	_, url, ok = split_once(url, "/")
	Assert(ok, "was not ok")

	// now it has a number, then maybe another slash and chapter name
	if strings.Contains(url, "/") {
		url = url[:strings.Index(url, "/")]
	}

	chapter_ident := url

	return Royal_Ident{
		fiction_ident: fiction_ident,
		chapter_ident: chapter_ident,
	}
}

func (r Royal_Ident) to_full_url() string {
	return fmt.Sprintf("https://www.royalroad.com/fiction/%s/chapter/%s", r.fiction_ident, r.chapter_ident)
}

func (r Royal_Ident) to_cannon_file_ident() string {
	return fmt.Sprintf("%s-%s", r.fiction_ident, r.chapter_ident)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

const DEBUG_PRINT_URL_OR_CACHE = true
const CACHED_FOLDER_NAME = "./cached/"

func get_url_or_cached(url string) string {
	royal_ident := parse_RoyalRoad_url_to_canonical(url)

	{ // make cached folder if it didn't exist
		exist, err := exists(CACHED_FOLDER_NAME)
		Assert(err == nil, err)
		if !exist {
			err := os.Mkdir(CACHED_FOLDER_NAME, 0755)
			Assert(err == nil, err)
		}
	}

	// check if the page is in the cache
	possible_filename := fmt.Sprintf("%s%s", CACHED_FOLDER_NAME, royal_ident.to_cannon_file_ident())
	exist, err := exists(possible_filename)
	Assert(err == nil, err)
	if exist {
		if DEBUG_PRINT_URL_OR_CACHE {
			fmt.Printf("Using cache!\n")
		}

		// read from the file
		bytes, err := os.ReadFile(possible_filename)
		Assert(err == nil, err)
		return string(bytes)

	} else {
		if DEBUG_PRINT_URL_OR_CACHE {
			fmt.Printf("URL is not cached. caching %s\n", url)
		}

		// download from the internet
		html := get_html_from_url(royal_ident.to_full_url())

		// cache the result
		err := dump_string(html, possible_filename)
		Assert(err == nil, err)
		return html
	}
}

func main() {
	url := "https://www.royalroad.com/fiction/72359/cartaflore/chapter/2059865/chapter-174-honest-red-reflection"

	body := get_url_or_cached(url)
	fmt.Printf("len body %d\n", len(body))

	html_doc, err := parse_HTML_Document(string(body))
	Assert(err == nil, err)

	element, err := find_element_by_header(html_doc, CHAPTER_INNER_CLASS)
	Assert(err == nil, err)

	out_put_markdown_text := strings.Builder{}

	{ // Deal with the Title
		title_text, err := find_element_by_header(html_doc, HEADER_CLASS)
		Assert(err == nil, err)
		out_put_markdown_text.WriteString("# ")
		out_put_markdown_text.WriteString(title_text.all_subtext)
		out_put_markdown_text.WriteString("\n")
	}

	{ // Do the Body of the chapter
		top_level := all_top_level_indices(html_doc, element)
		for _, i := range top_level {
			item := html_doc.all_elements[i]

			if item.class != "p" {
				fmt.Printf("not doing the div, its a trap at %d\n", i)
				continue
			}

			// fmt.Printf("%d -> %s\n", i, item.class)

			sub_elements := all_top_level_indices(html_doc, item)
			for _, sub_ele_i := range sub_elements {
				sub_ele := html_doc.all_elements[sub_ele_i]

				switch sub_ele.class {
				case "span":
					{
						out_put_markdown_text.WriteString(sub_ele.all_subtext)
					}
				case "em":
					{
						Assert(num_sub_elements(sub_ele) == 1, "em tag is italics, should only contain one class")
						out_put_markdown_text.WriteString("*")
						out_put_markdown_text.WriteString(html_doc.all_elements[sub_ele.own_index+1].all_subtext)
						out_put_markdown_text.WriteString("*")
					}
				default:
					{
						fmt.Printf("Unknown class found, %s\n", sub_ele.class)
					}
				}
			}

			out_put_markdown_text.WriteString("\n\n")
		}
	}

	result := out_put_markdown_text.String()
	// out_filename := fmt.Sprintf("%s/")
	dump_string(result, "test.md")

	print("Its all Good!\n")
}
