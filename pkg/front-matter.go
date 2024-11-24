package omh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/gernest/front"
	"gopkg.in/yaml.v2"
)

// Define errors for front matter parsing
var (
	ErrNoFrontMatter = errors.New("missing front matter")
)

// Global front matter handler
var frontMatter *front.Matter

// FrontMatter is meta information for markdown documents
type FrontMatter map[string]interface{}

// Checks if a specific key exists in the front matter
func (fm FrontMatter) Has(key string) bool {
	_, ok := fm[key]
	return ok
}

// Retrieves the value for a key as a string
func (fm FrontMatter) String(key string) string {
	v, ok := fm[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// Retrieves the value for a key as a slice of strings
func (fm FrontMatter) Strings(key string) []string {
	v, ok := fm[key]
	if !ok {
		return nil
	}
	ss, ok := v.([]string)
	if ok {
		return ss
	}
	ii, ok := v.([]interface{})
	if ok {
		ss = make([]string, len(ii))
		for i, vv := range ii {
			s, ok := vv.(string)
			if ok {
				ss[i] = s
			} else {
				ss[i] = fmt.Sprintf("%v", vv)
			}
		}
		return ss
	}
	return nil
}

// Parses a Markdown file into front matter and body content
func ParseFrontMatterMarkdown(content []byte) (FrontMatter, string, error) {
	metaLines := make([]string, 0) // Metadata lines
	bodyLines := make([]string, 0) // Markdown body lines
	state := 0                     // Tracks parsing state (0: before meta, 1: in meta, 2: in body)

	// Read content line by line
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		// Detect YAML front matter delimiters
		if state < 2 && line == "---" {
			state++
			continue
		}
		if state == 1 { // Inside front matter
			metaLines = append(metaLines, line)
		} else if state == 2 { // Inside body
			bodyLines = append(bodyLines, line)
		}
	}
	// Return an error if no metadata is found
	if len(metaLines) == 0 {
		return nil, "", ErrNoFrontMatter
	}

	// Parse metadata as YAML
	meta := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(strings.Join(metaLines, "\n")), &meta)
	if err != nil {
		return nil, "", err
	}

	// Return front matter, body content, and error (if any)
	return FrontMatter(meta), strings.TrimSpace(strings.Join(bodyLines, "\n")), nil
}

// Initializes the front matter parser
func init() {
	frontMatter = front.NewMatter()
	frontMatter.Handle("---", front.YAMLHandler) // Configure YAML front matter handling
}
