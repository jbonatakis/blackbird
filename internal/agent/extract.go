package agent

import (
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrNoJSONFound       = errors.New("no JSON object found in agent output")
	ErrMultipleJSONFound = errors.New("multiple JSON objects found in agent output")
)

func ExtractJSON(output string) (string, error) {
	if jsonBlock, ok, err := extractFencedJSON(output); err != nil {
		return "", err
	} else if ok {
		return jsonBlock, nil
	}

	objects := findJSONObjectCandidates(output)
	if len(objects) == 0 {
		return "", ErrNoJSONFound
	}
	if len(objects) > 1 {
		return "", ErrMultipleJSONFound
	}
	return objects[0], nil
}

func extractFencedJSON(output string) (string, bool, error) {
	type block struct {
		start int
		end   int
		body  string
	}
	var blocks []block

	search := output
	offset := 0
	for {
		idx := strings.Index(search, "```")
		if idx == -1 {
			break
		}
		open := offset + idx
		search = output[open+3:]
		offset = open + 3

		lineEnd := strings.IndexByte(search, '\n')
		if lineEnd == -1 {
			continue
		}
		lang := strings.TrimSpace(search[:lineEnd])
		bodyStart := offset + lineEnd + 1
		search = output[bodyStart:]
		offset = bodyStart

		closeIdx := strings.Index(search, "```")
		if closeIdx == -1 {
			continue
		}
		bodyEnd := offset + closeIdx
		body := output[bodyStart:bodyEnd]
		search = output[bodyEnd+3:]
		offset = bodyEnd + 3

		if strings.EqualFold(lang, "json") {
			blocks = append(blocks, block{start: bodyStart, end: bodyEnd, body: body})
		}
	}

	if len(blocks) == 0 {
		return "", false, nil
	}
	if len(blocks) > 1 {
		return "", true, ErrMultipleJSONFound
	}

	return strings.TrimSpace(blocks[0].body), true, nil
}

func findJSONObjectCandidates(output string) []string {
	var objs []string
	var start int
	inString := false
	escape := false
	depth := 0

	for i, r := range output {
		if escape {
			escape = false
			continue
		}
		if r == '\\' && inString {
			escape = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if r == '{' {
			if depth == 0 {
				start = i
			}
			depth++
			continue
		}
		if r == '}' {
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 {
				candidate := output[start : i+1]
				if jsonValidObject(candidate) {
					objs = append(objs, candidate)
				}
			}
		}
	}

	return objs
}

func jsonValidObject(candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	if !strings.HasPrefix(candidate, "{") || !strings.HasSuffix(candidate, "}") {
		return false
	}
	return json.Valid([]byte(candidate))
}
