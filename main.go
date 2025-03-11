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

var context_indent = 0

func log(format string, a ...any) {
	Assert(context_indent >= 0, "context indent was negative")

	for i := 0; i < context_indent; i++ {
		fmt.Printf(" ")
	}
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}

func dump_string(s string, filename string) error {
	f, err := os.Create(filename)
	Assert(err == nil, err)
	defer f.Close()
	_, err = f.WriteString(s)
	return err
}

func get_html_from_url(url string) string {
	log("Loading url: %s", url)

	resp, err := http.Get(url)
	Assert(err == nil, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	Assert(err == nil, err)

	log("finished loading: %s", url)

	return string(body)
}

func split_once(s string, sep string) (string, string, bool) {
	split := strings.SplitN(s, sep, 2)
	if len(split) == 1 {
		return s, "", false
	}
	return split[0], split[1], true
}

type RR_Fiction_Identifier string
type RR_Chapter_Identifier string

type Royal_Ident struct {
	fiction_ident RR_Fiction_Identifier
	chapter_ident RR_Chapter_Identifier
}

func parse_RoyalRoad_chapter_url_to_canonical(url string) Royal_Ident {
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
		fiction_ident: RR_Fiction_Identifier(fiction_ident),
		chapter_ident: RR_Chapter_Identifier(chapter_ident),
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

func make_folder_if_not_exists(name string) error {
	exist, err := exists(name)
	if err != nil || exist {
		return err
	}

	return os.Mkdir(name, 0755)
}

func get_url_or_cached(url string) string {
	royal_ident := parse_RoyalRoad_chapter_url_to_canonical(url)

	// make the cache if it doesn't exist
	err := make_folder_if_not_exists(CACHED_FOLDER_NAME)
	Assert(err == nil, err)

	// check if the page is in the cache
	possible_filename := fmt.Sprintf("%s%s.html", CACHED_FOLDER_NAME, royal_ident.to_cannon_file_ident())
	exist, err := exists(possible_filename)
	Assert(err == nil, err)
	if exist {
		if DEBUG_PRINT_URL_OR_CACHE {
			log("Using cache!")
		}

		// read from the file
		bytes, err := os.ReadFile(possible_filename)
		Assert(err == nil, err)
		return string(bytes)

	} else {
		if DEBUG_PRINT_URL_OR_CACHE {
			log("URL is not cached. caching %s", url)
		}

		// download from the internet
		html := get_html_from_url(royal_ident.to_full_url())

		// cache the result
		err := dump_string(html, possible_filename)
		Assert(err == nil, err)
		return html
	}
}

// have this take a Royal_Ident instead?
func rr_fiction_ident_to_list_of_chapters(fi RR_Fiction_Identifier) []string {
	url := "https://www.royalroad.com/fiction/" + string(fi)

	// we can't cache this because it changes all the time. and we want to get new updates
	body := get_html_from_url(url)

	doc, err := parse_HTML_Document(body)
	Assert(err == nil, err)

	const TABLE_BODY_TAG = "<tbody>"
	const CORRECT_TABLE_ELEMENT_ELEMENTS = 5

	table, err := find_element_by_header(doc, TABLE_BODY_TAG)
	Assert(err == nil, err)

	results := make([]string, 0)

	table_element_indexes := all_top_level_indices(doc, table)
	for _, t_index := range table_element_indexes {
		table_element := doc.all_elements[t_index]

		{ // check for errors
			Assert(table_element.class == "tr", "the elements don't look right wanted 'tr'")

			sub_elements := num_sub_elements(table_element)
			error_text := fmt.Sprintf("this table element dose not look right, has %d want %d", sub_elements, CORRECT_TABLE_ELEMENT_ELEMENTS)
			Assert(num_sub_elements(table_element) == CORRECT_TABLE_ELEMENT_ELEMENTS, error_text)
		}

		const START_DATA_URL = "data-url=\""

		data_url_index := strings.Index(table_element.heading_tag, START_DATA_URL)
		Assert(data_url_index != -1, "subtext must contain DATA_URL")

		start_of_url := table_element.heading_tag[data_url_index+len(START_DATA_URL):]
		Assert(start_of_url[0] == '/', "hope I moved right")

		end_of_string := strings.Index(start_of_url, "\"")
		Assert(data_url_index != -1, "missing end of string")

		data_url := start_of_url[:end_of_string]

		results = append(results, "https://www.royalroad.com"+data_url)
	}

	return results
}

func all_chapters_from_fiction_to_markdown(fiction RR_Fiction_Identifier) {
	log("loading info store")
	context_indent += 4
	rr_if, err := get_info_storage()
	Assert(err == nil, err)
	context_indent -= 4

	saved_indent := context_indent
	defer func() {
		context_indent = saved_indent
		save_info_storage(rr_if)
	}()

	log("getting the chapters")
	context_indent += 4
	chapters := rr_fiction_ident_to_list_of_chapters(fiction)
	context_indent -= 4

	log("turning chapters to markdown")
	context_indent += 4
	for i, chapter := range chapters {
		log("%03d/%03d -> %s", i+1, len(chapters), chapter)
		context_indent += 4

		body := get_url_or_cached(chapter)

		log("parsing html")
		context_indent += 4
		html_doc, err := parse_HTML_Document(body)
		Assert(err == nil, err)
		rr_chapter := Royal_Road_Chapter{
			original_url: chapter,
			doc:          html_doc,
		}
		context_indent -= 4

		rr_ident := parse_RoyalRoad_chapter_url_to_canonical(rr_chapter.original_url)

		context_indent += 4
		{ // set the rr_if info to the correct thing
			_, exist_fiction := rr_if.fiction_ident_to_titles[rr_ident.fiction_ident]
			if !exist_fiction {
				log("New Fiction found!")
				rr_if.fiction_ident_to_titles[rr_ident.fiction_ident] = rr_chapter.get_fiction_title_from_chapter()
			}

			_, exist_chapter := rr_if.chapter_ident_to_titles[rr_ident.chapter_ident]
			if !exist_chapter {
				log("New chapter found!")
				rr_if.chapter_ident_to_titles[rr_ident.chapter_ident] = rr_chapter.get_chapter_title()
			}
		}
		context_indent -= 4

		log("converting to markdown")
		context_indent += 4
		rr_chapter_markdown := rr_chapter.to_markdown()
		context_indent -= 4

		log("saving to file")
		context_indent += 4
		{ // put the markdown into its proper place

			const MARKDOWN_OUTPUT_FOLDER = "./markdown/"

			Assert(contains(rr_if.fiction_ident_to_titles, rr_ident.fiction_ident), "no fiction ident in rr_if, impossible")
			Assert(contains(rr_if.chapter_ident_to_titles, rr_ident.chapter_ident), "no chapter ident in rr_if, impossible")

			make_folder_if_not_exists(MARKDOWN_OUTPUT_FOLDER)
			make_folder_if_not_exists(MARKDOWN_OUTPUT_FOLDER + rr_if.fiction_ident_to_titles[rr_ident.fiction_ident])

			out_filename := fmt.Sprintf(MARKDOWN_OUTPUT_FOLDER+"%s/%s.md", rr_if.fiction_ident_to_titles[rr_ident.fiction_ident], rr_if.chapter_ident_to_titles[rr_ident.chapter_ident])
			dump_string(rr_chapter_markdown, out_filename)
		}
		context_indent -= 4

		context_indent -= 4
	}
	context_indent -= 4

	log("finished successfully!")
}

func main() {
	// const test_url = "https://www.royalroad.com/fiction/72359/cartaflore/chapter/2059865/chapter-174-honest-red-reflection"
	// const test_url = "https://www.royalroad.com/fiction/84669/heavenly-shae/chapter/2078862/manifold-journey-71-merchant-xio"
	// const test_url = "https://www.royalroad.com/fiction/69512/bog-standard-isekai/chapter/2077033/book-4-chapter-29"
	// const test_url = "https://www.royalroad.com/fiction/69512/bog-standard-isekai"
	// const test_url = "https://www.royalroad.com/fiction/107017/mage-lord-isekai"

	ident := RR_Fiction_Identifier("107017")
	all_chapters_from_fiction_to_markdown(ident)
}

func contains[T comparable, U any](m map[T]U, key T) bool {
	_, contains := m[key]
	return contains
}
