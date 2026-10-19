package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTodos(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Todo
	}{
		{
			name: "line todo with continuation",
			src: `package x
//TODO: first
//second
//third
func f() {}`,
			want: []Todo{{Text: "first second third", Line: 2}},
		},
		{
			name: "block todo with blank lines",
			src: `/*TODO:text

text

text

*/
code()`,
			want: []Todo{{Text: "text text text", Line: 1}},
		},
		{
			name: "single line block",
			src:  `x := 1 /* TODO: inline block */ + 2`,
			want: []Todo{{Text: "inline block", Line: 1}},
		},
		{
			name: "javadoc style stars",
			src: `/** TODO: refactor
 * this method
 */`,
			want: []Todo{{Text: "refactor this method", Line: 1}},
		},
		{
			name: "trailing comment and indented continuation",
			src: `	x := 1 // TODO: fix this
	// and that
	y := 2`,
			want: []Todo{{Text: "fix this and that", Line: 1}},
		},
		{
			name: "consecutive todos are separate",
			src: `// TODO: one
// TODO: two
// more two`,
			want: []Todo{{Text: "one", Line: 1}, {Text: "two more two", Line: 2}},
		},
		{
			name: "empty comment ends todo",
			src: `// TODO: one
//
// unrelated`,
			want: []Todo{{Text: "one", Line: 1}},
		},
		{
			name: "not todos",
			src: `url := "http://example.com/TODO"
glob := "src/*.go"
// todo lowercase is ignored
// TODOS is not a marker
/* regular comment */`,
			want: nil,
		},
		{
			name: "todo without colon",
			src:  `// TODO handle errors`,
			want: []Todo{{Text: "handle errors", Line: 1}},
		},
		{
			name: "unterminated block at EOF",
			src: `/* TODO: never closed
still text`,
			want: []Todo{{Text: "never closed still text", Line: 1}},
		},
		{
			name: "block comment after line todo continues it",
			src: `	//TODO: refactor
	/*
		defer is bound to the function,
		not the block
	*/
	code()`,
			want: []Todo{{Text: "refactor defer is bound to the function, not the block", Line: 1}},
		},
		{
			name: "single line block after line todo",
			src: `// TODO: a
// b
/* c */ // TODO: d`,
			want: []Todo{{Text: "a b c", Line: 1}, {Text: "d", Line: 3}},
		},
		{
			name: "block todo after line todo is separate",
			src: `// TODO: a
/* TODO: b */`,
			want: []Todo{{Text: "a", Line: 1}, {Text: "b", Line: 2}},
		},
		{
			name: "todo after block end on same line",
			src: `/* TODO: a
b */ // TODO: c`,
			want: []Todo{{Text: "a b", Line: 1}, {Text: "c", Line: 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTodos(strings.NewReader(tt.src), "f.go")
			if err != nil {
				t.Fatal(err)
			}
			for i := range tt.want {
				tt.want[i].File = "f.go"
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
