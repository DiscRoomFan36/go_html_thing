package main

import (
	"fmt"
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

// func get_html_from_url(url string) string {
// 	resp, err := http.Get(url)
// 	Assert(err == nil, err)
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	Assert(err == nil, err)

// 	return string(body)
// }

const CHAPTER_INNER_CLASS = "<div class=\"chapter-inner chapter-content\">"

func num_sub_elements(sub HTML_SubClass) int {
	return int(sub.final_index) - int(sub.own_index) - 1
}

func all_top_level_indices(doc HTML_Document, element HTML_SubClass) []int {
	results := make([]int, 0)
	i := element.own_index + 1
	for i < uint64(element.final_index) {
		results = append(results, int(i))
		i = uint64(doc.all_elements[i].final_index)
	}
	return results
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

	text_holder := -1
	for index, inner := range html_doc.all_elements {
		// fmt.Printf("%d (%s) -> %d\n", index, inner.class, inner.final_index)
		if inner.heading_tag == CHAPTER_INNER_CLASS {
			text_holder = index
			break
		}
	}

	Assert(text_holder != -1, "did not find it")

	element := html_doc.all_elements[text_holder]

	// TODO debug subtext... the inner is to bing
	// fmt.Printf("subtext length: %d\n", len(element.all_subtext))
	// fmt.Printf("sub_elements:   %d\n", num_sub_elements(element))
	// fmt.Printf("%+v\n", element)

	// for i := element.own_index + 1; i < uint64(element.final_index); i++ {
	// 	item := html_doc.all_elements[i]
	// 	if item.class == "p" {
	// 		fmt.Printf("%d -> %d\n", i, num_sub_elements(item))
	// 		if num_sub_elements(item) > 1 {
	// 			fmt.Printf("{%s}\n", item.all_subtext)
	// 		}
	// 	}
	// 	if item.class == "div" {
	// 		fmt.Printf("div -> {%s}\n", item.all_subtext)
	// 	}
	// 	// if item.own_index+1 == uint64(item.final_index) {
	// 	// 	fmt.Printf("inner -> {%s}\n", item.all_subtext)
	// 	// }
	// }

	out_put_markdown_text := strings.Builder{}

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

			// Unneeded?
			// out_put_markdown_text.WriteString("{Cowabunga}")
		}

		out_put_markdown_text.WriteString("\n\n")
	}

	result := out_put_markdown_text.String()
	dump_string(result, "test.md")

	print("Its all Good!\n")
}
