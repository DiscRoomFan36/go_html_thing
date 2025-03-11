package main

import (
	"errors"
	"strings"
)

func Assert(b bool, reason any) {
	if !b {
		panic(reason)
	}
}

type HTML_Document struct {
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

func is_alpha(c byte) bool {
	if 'a' <= c && c <= 'z' {
		return true
	}
	if 'A' <= c && c <= 'Z' {
		return true
	}

	return false
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
	switch class_tag {
	case "meta":
		return true
	case "link":
		return true
	case "img":
		return true
	case "input":
		return true
	case "br": // NOTE <br> is actually a line break...
		return true
	case "hr": // NOTE <hr> is some header thing...
		return true
	case "partial":
		return true
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
		// log("{%c} -> {%s}", document[j], document[j-5:j+5])
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
				new_helper.sub.final_index = int64(len(html_doc.all_elements))
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
			// log("class tag: %s", class_tag)
			// log("last  tag: %s", subclass_stack[len(subclass_stack)-1].sub.class)

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

				// log("start_text_index: %d", item.start_text_index)

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

type Royal_Road_Chapter struct {
	original_url string

	doc HTML_Document
}

func (rr_chapter Royal_Road_Chapter) is_royal_road_link() bool {
	if rr_chapter.original_url == "" {
		panic("did not set url before calling this function!!!")
	}

	if strings.HasPrefix(rr_chapter.original_url, "https://www.royalroad.com") {
		return true
	}
	return false
}

func (rr_chapter Royal_Road_Chapter) is_rr_chapter() bool {
	if rr_chapter.original_url == "" {
		panic("did not set url before calling this function!!!")
	}

	if !rr_chapter.is_royal_road_link() {
		return false
	}

	_, right, ok := split_once(rr_chapter.doc.original_string, "chapter/")
	if ok {
		Assert(len(right) > 0, "invalid link")
		return true
	}

	return false
}

func (rr_chapter Royal_Road_Chapter) get_chapter_title() string {
	Assert(rr_chapter.is_rr_chapter(), "HTML must be a royal road chapter link")

	// royal road header class. might break...
	const HEADER_CLASS = "<h1 style=\"margin-top: 10px\" class=\"font-white break-word\">"
	title_text, err := find_element_by_header(rr_chapter.doc, HEADER_CLASS)
	Assert(err == nil, err)

	return title_text.all_subtext
}

func (rr_chapter Royal_Road_Chapter) get_fiction_title_from_chapter() string {
	Assert(rr_chapter.is_rr_chapter(), "HTML must be a royal road chapter link")

	const FICTION_TITLE_CLASS = "<h2 style=\"font-size: 24px\" class=\"font-white inline-block\">"

	title_text, err := find_element_by_header(rr_chapter.doc, FICTION_TITLE_CLASS)
	Assert(err == nil, err)

	return title_text.all_subtext
}

func reverse[T any](array []T) {
	for i := 0; i < len(array)/2; i++ {
		j := len(array) - 1 - i
		array[i], array[j] = array[j], array[i]
	}
}

func (rr_chapter Royal_Road_Chapter) to_markdown() string {
	const CHAPTER_INNER_CLASS = "<div class=\"chapter-inner chapter-content\">"

	Assert(rr_chapter.is_rr_chapter(), "must be a rr chapter")

	doc := rr_chapter.doc

	output_markdown_text := strings.Builder{}

	{ // Deal with the Title
		output_markdown_text.WriteString("# ")
		output_markdown_text.WriteString(rr_chapter.get_chapter_title())
		output_markdown_text.WriteString("\n")
	}

	{ // Do the Body of the chapter
		element, err := find_element_by_header(doc, CHAPTER_INNER_CLASS)
		Assert(err == nil, err)

		// TODO some fictions bury the <p> tags in a nest of <div>'s???
		// check number of sub elements, and if more than 3 or something, do it...

		// were gonna treat this as a stack, to handle some weird case
		// where we need to add more elements to this array
		top_level_indices := all_top_level_indices(doc, element)
		reverse(top_level_indices)

		var i int
		for len(top_level_indices) > 0 {
			top_level_indices, i = pop(top_level_indices)
			// for _, i := range top_level_indices {

			item := doc.all_elements[i]

			// log("item.class %s, len %d", item.class, num_sub_elements(item))

			if item.class != "p" {
				log("theres a non <p> block at %d, skipping", i)

				if num_sub_elements(item) > 5 {
					log("actually... this entry seems fishy, going deeper")

					// something fishy is going on
					// we want to print this new thing...
					// but beware of the order your doing things, we want this to happen first

					// get child classes
					fishy_elements := all_top_level_indices(doc, item)
					// reverse to match the 'top_level_indices' array
					reverse(fishy_elements)

					top_level_indices = append(top_level_indices, fishy_elements...)
				}

				continue
			}

			// log("%d -> %s", i, item.class)

			html_subclass_to_markdown_text(doc, item, &output_markdown_text)

			// this is incorrect... missing things not in spans... move more into thing below and remove recursion
			// for _, sub_ele_i := range all_top_level_indices(doc, item) {
			// 	sub_ele := doc.all_elements[sub_ele_i]

			// 	html_subclass_to_markdown_text(doc, sub_ele, &output_markdown_text)
			// }

			output_markdown_text.WriteString("\n\n")
		}
	}

	return output_markdown_text.String()
}

// turn "</p class='bobber'>" into "p"
func get_class_name_from_heading_tag(tag string) string {
	Assert(tag[0] == '<', "must be valid tag")

	i := 1
	if tag[i] == '/' {
		i += 1
	}
	start := i

	for i < len(tag) && is_alpha(tag[i]) {
		i += 1
	}
	Assert(i < len(tag), "must be a valid tag, ran of the edge")

	return tag[start:i]
}

func get_all_un_tagged_text_in_all_subtext(sub HTML_SubClass) []string {
	result := make([]string, 0)

	base := 0
	index := 0

	for {
		for index < len(sub.all_subtext) && sub.all_subtext[index] != '<' {
			index += 1
		}

		// log("got a thing: |%s|", sub.all_subtext[index-5:index])

		result = append(result, sub.all_subtext[base:index])

		if index >= len(sub.all_subtext) {
			// this means were past the end of the subtext, and can go home...
			break
		}

		tag_start := index

		// go past the <tag>
		for index < len(sub.all_subtext) && sub.all_subtext[index] != '>' {
			index += 1
		}
		Assert(index < len(sub.all_subtext), "must be true because parse_HTML was already run")

		// check if this thing is one of the dumb ones...
		tag_class := get_class_name_from_heading_tag(sub.all_subtext[tag_start : index+1])
		if class_tag_is_one_of_the_dumb_ones(tag_class) {
			// we can skip the thing
			continue
		}

		// for dumb recursive text... should have just hade the HTML thing handle all this...
		depth := 1

		// now find the end of this block, while respecting sub tags of the same name...
		for index < len(sub.all_subtext) {
			for index < len(sub.all_subtext) && sub.all_subtext[index] != '<' {
				index += 1
			}
			Assert(index < len(sub.all_subtext), "must be true because parse_HTML was already run")

			end_tag := "</" + tag_class + ">"
			recur_start_tag := "<" + tag_class

			if sub.all_subtext[index:index+len(end_tag)] == end_tag {
				// thats the end of the good tag
				depth -= 1
				index += len(end_tag)
				if depth == 0 {
					break
				}
			} else if sub.all_subtext[index:index+len(recur_start_tag)] == recur_start_tag {
				depth += 1
				index += len(recur_start_tag)
				continue
			}
			index += 1
		}

		// Assert(index < len(sub.all_subtext), "i hope this doesn't break everything... maybe make this outer loop a while true loop")
		// advance the base
		base = index
	}

	return result
}

func html_subclass_to_markdown_text(doc HTML_Document, sub HTML_SubClass, output *strings.Builder) {
	// TODO handle dumb html things like "&nbsp;"
	// TODO remove recursion. i just don't like recursion that much

	switch sub.class {
	case "p":
		// the p element can hold a-lot of text, or no text...
		// were doing some kinda silly things to pick up on that
		// but it works! maybe!

		true_text := get_all_un_tagged_text_in_all_subtext(sub)
		sub_elements_indices := all_top_level_indices(doc, sub)
		Assert(len(true_text) == len(sub_elements_indices)+1, "this is how this thing should work...")

		output.WriteString(true_text[0])
		for i, sub_ele_i := range sub_elements_indices {
			sub_ele := doc.all_elements[sub_ele_i]
			html_subclass_to_markdown_text(doc, sub_ele, output)

			output.WriteString(true_text[i+1])
		}

	case "span":
		Assert(num_sub_elements(sub) == 0, "spans can only contain text...")
		output.WriteString(sub.all_subtext)

	case "em":
		output.WriteString("*")
		if num_sub_elements(sub) == 0 {
			output.WriteString(sub.all_subtext)
		} else {
			html_subclass_to_markdown_text(doc, doc.all_elements[sub.own_index+1], output)
		}
		output.WriteString("*")

	case "strong":
		output.WriteString("**")
		if num_sub_elements(sub) == 0 {
			output.WriteString(sub.all_subtext)
		} else {
			html_subclass_to_markdown_text(doc, doc.all_elements[sub.own_index+1], output)
		}
		output.WriteString("**")

	default:
		log("Unknown class found, %s", sub.class)
	}
}
