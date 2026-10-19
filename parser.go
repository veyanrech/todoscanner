package main

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// Todo is a single TODO found in a source file.
type Todo struct {
	Text string
	File string
	Line int
}

// todoRe matches the TODO marker at the start of comment text, e.g. "TODO:", " TODO ".
var todoRe = regexp.MustCompile(`^\s*TODO\b:?\s*`)

type parseState int

const (
	stateNone  parseState = iota
	stateLine             // inside a //TODO, collecting following // lines
	stateBlock            // inside a /*TODO, collecting until */
)

const maxLineSize = 16 * 1024 * 1024

// ParseTodos reads source code and returns all TODOs in it.
//
// Supported forms:
//
//	//TODO: text          followed by any number of "// text" lines
//	/*TODO: text ... */   everything up to the closing */
//
// A // TODO ends at the first line that is not a // comment, is an empty
// // comment, or starts another TODO. A /* */ comment directly after it is
// appended to it.
func ParseTodos(r io.Reader, path string) ([]Todo, error) {
	var (
		todos  []Todo
		state  = stateNone
		cur    Todo
		parts  []string
		lineNo int
	)

	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			parts = append(parts, s)
		}
	}
	flush := func() {
		cur.Text = strings.Join(parts, " ")
		todos = append(todos, cur)
		parts = nil
		state = stateNone
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), maxLineSize)
	for sc.Scan() {
		lineNo++
		line := sc.Text()

		switch state {
		case stateBlock:
			end := strings.Index(line, "*/")
			if end < 0 {
				add(cleanBlockLine(line))
				continue
			}
			add(cleanBlockLine(line[:end]))
			flush()
			// The rest of the line may start another TODO.
			line = line[end+2:]

		case stateLine:
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				body := strings.TrimLeft(trimmed, "/")
				if strings.TrimSpace(body) != "" && !todoRe.MatchString(body) {
					add(body)
					continue
				}
			}
			// A /* */ comment right after a //TODO continues it.
			if body, ok := strings.CutPrefix(trimmed, "/*"); ok {
				body = strings.TrimLeft(body, "*")
				if !todoRe.MatchString(body) {
					end := strings.Index(body, "*/")
					if end < 0 {
						add(cleanBlockLine(body))
						state = stateBlock
						continue
					}
					add(body[:end])
					line = body[end+2:]
				}
			}
			flush()
		}

		// Look for the start of a new TODO on this line.
		for i := 0; i+1 < len(line); i++ {
			if line[i] != '/' || (line[i+1] != '/' && line[i+1] != '*') {
				continue
			}
			isBlock := line[i+1] == '*'
			rest := line[i+2:]
			if isBlock {
				rest = strings.TrimLeft(rest, "*") // allow /**TODO
			}
			loc := todoRe.FindStringIndex(rest)
			if loc == nil {
				continue
			}
			text := rest[loc[1]:]
			cur = Todo{File: path, Line: lineNo}

			if !isBlock {
				add(text)
				state = stateLine
				break
			}
			if end := strings.Index(text, "*/"); end >= 0 {
				add(text[:end])
				flush()
				break
			}
			add(text)
			state = stateBlock
			break
		}
	}
	if err := sc.Err(); err != nil {
		return todos, err
	}
	if state != stateNone {
		flush()
	}
	return todos, nil
}

// cleanBlockLine strips whitespace and a leading "*" used in /* ... */ comments.
func cleanBlockLine(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "*")
	return strings.TrimSpace(s)
}
