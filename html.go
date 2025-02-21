package main

import (
	"errors"
	"fmt"
	"strings"
)

func Assert(b bool, reason any) {
	if !b {
		panic(reason)
	}
}

type HTML_Document struct {
	// this is not set in the parse HTML function,
	// please set it at your own convenience
	original_url string

	// contains the bytes or string that is the entire HTML doc
	original_string string

	// all of the elements stored in one place instead of on the stack
	all_elements []HTML_SubClass

	// the main headings? unnecessary?
	// root_nodes []uint64
}

type HTML_SubClass struct {
	// string slice into original string
	heading_tag string
	// slice into the heading tag (will be "div" or something)
	class string

	// which item is this in the big list. helpful to compute parent_index
	own_index uint64
	// which item in all_elements owns this. could be -1
	parent_index int64

	// slice into the original_string, contains all of the inner text, and subclasses
	all_subtext string

	// // index into the all_elements of the HTML_Document, in order
	// sub_elements []uint64

	// the last sub element, could use this to replace the above array, and just be sick nasty with my array management
	// could be -1, for testing purpose
	// "up-to but not including"
	final_index int64
}

func is_alphanumeric(c byte) bool {
	if 'a' <= c && c <= 'z' {
		return true
	}
	if 'A' <= c && c <= 'Z' {
		return true
	}
	if '0' <= c && c <= '9' {
		return true
	}

	return false
}

// this is html only?: https://infra.spec.whatwg.org/#ascii-whitespace
func is_whitespace(c byte) bool {
	if c == ' ' {
		return true
	}
	if c == '\t' {
		return true
	}
	if c == 0x0A /* LF */ {
		return true
	}
	if c == 0x0C /* FF */ {
		return true
	}
	if c == 0x0D /* CR */ {
		return true
	}

	return false
}

func pop[T any](array []T) ([]T, T) {
	item := array[len(array)-1]
	array = array[:len(array)-1]
	return array, item
}

func class_tag_is_one_of_the_dumb_ones(class_tag string) bool {
	// NOTE <br> is actually a line break...
	// NOTE <hr> is some header thing...
	var AUTO_VOID_TAGS = [...]string{"meta", "link", "img", "input", "br", "hr"}
	for _, tag := range AUTO_VOID_TAGS {
		if tag == class_tag {
			return true
		}
	}
	return false
}

func parse_HTML_Document(document string) (HTML_Document, error) {
	html_doc := HTML_Document{
		original_string: document,
		all_elements:    make([]HTML_SubClass, 0, 64),
		// root_nodes:      make([]uint64, 0, 8),
	}

	type helper_struct struct {
		sub              *HTML_SubClass
		start_text_index int
	}

	subclass_stack := make([]helper_struct, 0, 64)

	// find the next <
	index := 0
	for {
		for index < len(document) && document[index] != '<' {
			index += 1
		}

		// check if were done parsing
		if index >= len(document) {
			break
		}

		Assert(document[index] == '<', "its gotta")

		// check that its not a comment
		const COMMENT_SYM = "<!--"
		Assert(len(document) > index+len(COMMENT_SYM), "must be long enough for this. TODO?")
		if document[index:index+len(COMMENT_SYM)] == COMMENT_SYM {
			// skip the comment, and move forward enough to account for <!-->
			j := index + len(COMMENT_SYM) + 3
			for j < len(document) {
				if document[j] == '>' {
					// move backward to check
					if document[j-1] == '-' && document[j-2] == '-' {
						// end of comment
						break
					}
				}

				j += 1
			}

			Assert(j < len(document), "Bad comment")
			index = j + 1
			// now go back up to the top
			continue
		}

		// its not a comment parse <div /> or whatever
		j := index + 1
		Assert(j < len(document), "malformed div")

		// 0 is false, 1 is true, not bool for '/' reasons
		is_end_tag := false
		is_void_element := false
		if document[j] == '/' {
			is_end_tag = true
			j += 1
		} else if document[j] == '!' {
			is_void_element = true
			j += 1
		}

		Assert(j < len(document), "malformed div")

		// move past tag
		for j < len(document) && is_alphanumeric(document[j]) {
			j += 1
		}
		Assert(j < len(document), "malformed div")
		// fmt.Printf("{%c} -> {%s}\n", document[j], document[j-5:j+5])
		Assert(is_whitespace(document[j]) || document[j] == '>' || document[j] == '/', "must be followed by whitespace or '>' OR '/'")

		heading_base := index + 1
		if is_end_tag {
			heading_base += 1
		}
		class_tag := document[heading_base:j]

		// now parse the rest of the tag, and find the '>'
		k := j
		for k < len(document) {
			if document[k] == '>' {
				break
			}

			// NOTE you cannot put '"' in a string in HTML.
			// aka. <div text="\"hello\""> dose not work

			k += 1
		}
		Assert(k < len(document), "malformed div")

		// check last for /
		if document[k-1] == '/' {
			is_void_element = true // void element have no body
		}

		// we have all the info now...
		if !is_end_tag {
			var own_index uint64 = uint64(len(html_doc.all_elements))
			var parent_index int64 = -1
			if len(subclass_stack) != 0 {
				parent_index = int64(subclass_stack[len(subclass_stack)-1].sub.own_index)
			}

			html_doc.all_elements = append(html_doc.all_elements, HTML_SubClass{
				heading_tag:  document[index : k+1],
				class:        class_tag,
				own_index:    own_index,
				parent_index: parent_index,

				final_index: -1,
			})

			new_helper := helper_struct{
				sub:              &html_doc.all_elements[len(html_doc.all_elements)-1],
				start_text_index: k + 1,
			}

			if !is_void_element {
				if class_tag_is_one_of_the_dumb_ones(class_tag) {
					is_void_element = true
				}
			}

			if !is_void_element {
				subclass_stack = append(subclass_stack, new_helper)
			} else {
				new_helper.sub.all_subtext = document[k+1 : k+1]
				new_helper.sub.final_index = int64(len(subclass_stack))
			}

			if class_tag == "script" {
				Assert(!is_void_element, "if this breaks WTF")

				// we need to handle this...
				l := k + 1
				const END_SCRIPT_TAG = "</script>"
				for l < len(document) {
					if document[l] == '<' {
						Assert(l+len(END_SCRIPT_TAG) < len(document), "malformed script tag")
						if document[l:l+len(END_SCRIPT_TAG)] == END_SCRIPT_TAG {
							// we found the end
							break
						}
					}

					l += 1
				}
				Assert(l < len(document), "malformed script tag")
				Assert(document[l] == '<', "i think were here")

				var item helper_struct
				subclass_stack, item = pop(subclass_stack)
				item.sub.final_index = int64(len(html_doc.all_elements))
				item.sub.all_subtext = document[k+1 : l]

				k = l + len(END_SCRIPT_TAG) - 1
				Assert(document[k] == '>', "this is a real assert")
			}

		} else {
			// fmt.Printf("class tag: %s\n", class_tag)
			// fmt.Printf("last  tag: %s\n", subclass_stack[len(subclass_stack)-1].sub.class)

			if class_tag_is_one_of_the_dumb_ones(class_tag) {
				// HTML Sucks Ass
			} else {
				Assert(class_tag == subclass_stack[len(subclass_stack)-1].sub.class, "Closing class must be the same as the one who opened it.")

				// pop last item of the stack
				var item helper_struct
				subclass_stack, item = pop(subclass_stack)

				// TODO this is dumb, just store the position in the struct...
				// go pointers are stupid...
				real_item := &html_doc.all_elements[item.sub.own_index]

				// fmt.Printf("start_text_index: %d\n", item.start_text_index)

				real_item.final_index = int64(len(html_doc.all_elements))
				real_item.all_subtext = document[item.start_text_index:index]
			}
		}

		index = k + 1
	}

	Assert(len(subclass_stack) == 0, "malformed div")

	for _, item := range html_doc.all_elements {
		Assert(num_sub_elements(item) >= 0, "must have positive sub elements")
	}

	return html_doc, nil
}

// -------------------------------------
//              Helpers
// -------------------------------------

func num_sub_elements(sub HTML_SubClass) int {
	return int(sub.final_index) - int(sub.own_index) - 1
}

func find_element_by_header(doc HTML_Document, header string) (HTML_SubClass, error) {
	for _, inner := range doc.all_elements {
		if inner.heading_tag == header {
			return inner, nil
		}
	}
	return HTML_SubClass{}, errors.New("no element found")
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

// -------------------------------------
//           Royal Road Stuff
// -------------------------------------

func (doc HTML_Document) is_royal_road_link() bool {
	if doc.original_url == "" {
		panic("did not set url before calling this function!!!")
	}

	if strings.HasPrefix(doc.original_url, "https://www.royalroad.com") {
		return true
	}
	return false
}

func (doc HTML_Document) is_rr_chapter() bool {
	if doc.original_url == "" {
		panic("did not set url before calling this function!!!")
	}

	if !doc.is_royal_road_link() {
		return false
	}

	_, right, ok := split_once(doc.original_string, "chapter/")
	if ok {
		Assert(len(right) > 0, "invalid link")
		return true
	}

	return false
}

func (doc HTML_Document) get_chapter_title() string {
	Assert(doc.is_rr_chapter(), "HTML must be a royal road chapter link")

	// royal road header class. might break...
	const HEADER_CLASS = "<h1 style=\"margin-top: 10px\" class=\"font-white break-word\">"
	title_text, err := find_element_by_header(doc, HEADER_CLASS)
	Assert(err == nil, err)

	return title_text.all_subtext
}

func (doc HTML_Document) get_fiction_title_from_chapter() string {
	Assert(doc.is_rr_chapter(), "HTML must be a royal road chapter link")

	const FICTION_TITLE_CLASS = "<h2 style=\"font-size: 24px\" class=\"font-white inline-block\">"

	title_text, err := find_element_by_header(doc, FICTION_TITLE_CLASS)
	Assert(err == nil, err)

	return title_text.all_subtext
}

func (doc HTML_Document) rr_chapter_to_markdown() string {
	const CHAPTER_INNER_CLASS = "<div class=\"chapter-inner chapter-content\">"

	Assert(doc.is_rr_chapter(), "must be a rr chapter")

	out_put_markdown_text := strings.Builder{}

	{ // Deal with the Title
		out_put_markdown_text.WriteString("# ")
		out_put_markdown_text.WriteString(doc.get_chapter_title())
		out_put_markdown_text.WriteString("\n")
	}

	{ // Do the Body of the chapter
		element, err := find_element_by_header(doc, CHAPTER_INNER_CLASS)
		Assert(err == nil, err)

		for _, i := range all_top_level_indices(doc, element) {
			item := doc.all_elements[i]

			fmt.Printf("item.class %s, len %d\n", item.class, num_sub_elements(item))

			if item.class != "p" {
				fmt.Printf("theres a non <p> block at %d, skipping\n", i)
				continue
			}

			// fmt.Printf("%d -> %s\n", i, item.class)

			for _, sub_ele_i := range all_top_level_indices(doc, item) {
				sub_ele := doc.all_elements[sub_ele_i]

				switch sub_ele.class {
				case "span":
					{
						out_put_markdown_text.WriteString(sub_ele.all_subtext)
					}
				case "em":
					{
						Assert(num_sub_elements(sub_ele) == 1, "em tag is italics, should only contain one class")
						out_put_markdown_text.WriteString("*")
						out_put_markdown_text.WriteString(doc.all_elements[sub_ele.own_index+1].all_subtext)
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

	return out_put_markdown_text.String()
}
