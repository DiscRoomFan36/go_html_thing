package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const DEBUG_PRINT_EASY_PARSE_FILE = true

const RR_INFO_STORE_FILEPATH = "./RR.info_store" // TODO rename
const RR_IF_FILE_VERSION_NUMBER = 1

const PARSER_SEPARATOR_FIRST = " : "
const PARSER_SEPARATOR_SECOND = " -> "
const PARSER_FICTION_TAG = "FICTION"
const PARSER_CHAPTER_TAG = "CHAPTER"

type RoyalRoad_Info_Storage struct {
	fiction_ident_to_titles map[string]string

	// i think the chapters are unique as well...
	chapter_ident_to_titles map[string]string
}

func get_info_storage() (RoyalRoad_Info_Storage, error) {
	if DEBUG_PRINT_EASY_PARSE_FILE {
		fmt.Printf("Getting info storage\n")
	}

	result := RoyalRoad_Info_Storage{
		fiction_ident_to_titles: make(map[string]string),
		chapter_ident_to_titles: make(map[string]string),
	}

	{ // if the file dose not exist, create a empty file
		exist, err := exists(RR_INFO_STORE_FILEPATH)
		if err != nil {
			return result, err
		}
		if !exist {
			err := os.WriteFile(RR_INFO_STORE_FILEPATH, []byte("[1] # Version\n"), 0644)
			if err != nil {
				return result, err
			}
		}
	}

	// read the file and parse it
	data, err := os.ReadFile(RR_INFO_STORE_FILEPATH)
	if err != nil {
		return result, err
	}

	lines := strings.Split(string(data), "\n")

	strip_line_of_comments := func(line string) string {
		left, _, _ := split_once(line, "#")
		return strings.TrimRightFunc(left, unicode.IsSpace)
	}

	{ // parse the version
		version := strip_line_of_comments(lines[0])
		if version[0] != '[' {
			return result, errors.New("error parsing version, no '[' at start of file")
		}

		if version[len(version)-1] != ']' {
			return result, errors.New("error parsing version, no ']' at end of first line")
		}
		version_no_text := version[1 : len(version)-1]

		// assert that it only contains numbers
		for _, r := range version_no_text {
			if !unicode.IsDigit(r) {
				return result, errors.New("error parsing version, non digit char in brackets")
			}
		}

		version_number, err := strconv.ParseInt(version_no_text, 10, 0)
		if err != nil {
			return result, err
		}

		if version_number != RR_IF_FILE_VERSION_NUMBER {
			return result, fmt.Errorf("unknown version number, wanted %d got %d", RR_IF_FILE_VERSION_NUMBER, version_number)
		}
	}

	{ // now parse the rest of the file
		for index, line_w_comments := range lines[1:] {
			line_number := index + 2 // one for zero index, one for skipping version number

			line := strip_line_of_comments(line_w_comments)
			if line == "" {
				continue // skip empty lines
			}

			// format is:
			// FICTION : 912489 -> cool title
			selection, rest, ok := split_once(line, PARSER_SEPARATOR_FIRST)
			if !ok {
				error_text := fmt.Sprintf("%s:%d: no separator '%s' found on non empty line: {%s}", RR_INFO_STORE_FILEPATH, line_number, PARSER_SEPARATOR_FIRST, line)
				return result, errors.New(error_text)
			}

			ident, name, ok := split_once(rest, PARSER_SEPARATOR_SECOND)
			if !ok {
				error_text := fmt.Sprintf("%s:%d: no separator '%s' found on non empty line: {%s}", RR_INFO_STORE_FILEPATH, line_number, PARSER_SEPARATOR_SECOND, line)
				return result, errors.New(error_text)
			}

			switch selection {
			case PARSER_FICTION_TAG:
				result.fiction_ident_to_titles[ident] = name
			case PARSER_CHAPTER_TAG:
				result.chapter_ident_to_titles[ident] = name

			default:
				error_text := fmt.Sprintf("%s:%d: unknown selection '%s'found on line: {%s}", RR_INFO_STORE_FILEPATH, line_number, selection, line)
				return result, errors.New(error_text)
			}
		}
	}

	return result, nil
}

func save_info_storage(rr_if RoyalRoad_Info_Storage) error {
	if DEBUG_PRINT_EASY_PARSE_FILE {
		fmt.Printf("Saving info storage\n")
	}

	data := strings.Builder{}

	data.WriteString("[")
	data.WriteString(fmt.Sprintf("%d", RR_IF_FILE_VERSION_NUMBER))
	data.WriteString("] # Version\n\n")

	for ident, name := range rr_if.fiction_ident_to_titles {
		data.WriteString(PARSER_FICTION_TAG + PARSER_SEPARATOR_FIRST)
		data.WriteString(ident)
		data.WriteString(PARSER_SEPARATOR_SECOND)
		data.WriteString(name)
		data.WriteString("\n")
	}

	data.WriteString("\n")

	for ident, name := range rr_if.chapter_ident_to_titles {
		data.WriteString(PARSER_CHAPTER_TAG + PARSER_SEPARATOR_FIRST)
		data.WriteString(ident)
		data.WriteString(PARSER_SEPARATOR_SECOND)
		data.WriteString(name)
		data.WriteString("\n")
	}

	to_write := data.String()

	return os.WriteFile(RR_INFO_STORE_FILEPATH, []byte(to_write), 0644)
}
