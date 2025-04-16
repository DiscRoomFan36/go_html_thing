// -------------------------------------
//           Royal Road Stuff
// -------------------------------------

package main

import "strings"

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
		// Assert(num_sub_elements(sub) == 0, "spans can only contain text...", sub)
		if num_sub_elements(sub) != 0 {
			log("span contains %d sub elements, lets hope their only <br's>", num_sub_elements(sub))
		}
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
