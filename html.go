package main

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
	var AUTO_VOID_TAGS = [...]string{"meta", "link", "img", "input"}
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

			sub := HTML_SubClass{
				heading_tag:  document[index : k+1],
				class:        class_tag,
				own_index:    own_index,
				parent_index: parent_index,
			}

			html_doc.all_elements = append(html_doc.all_elements, sub)

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
				sub.all_subtext = document[k+1 : k+1]
				sub.final_index = int64(len(subclass_stack))
			}

			if class_tag == "script" {
				Assert(is_void_element == false, "if this breaks WTF")

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
				item.sub.final_index = int64(len(html_doc.all_elements))
				item.sub.all_subtext = document[item.start_text_index:index]
			}
		}

		index = k + 1
	}

	Assert(len(subclass_stack) == 0, "malformed div")

	return html_doc, nil
}
