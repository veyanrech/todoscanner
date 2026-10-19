# Purpose

CLI tool to parse the codebase (-f /path/to/code) to find all TODOs

Generates the output .md file with the list of TODOs

What is proper TODO:

//TODO: text
//text
//text

or

/*TODO:text

text

text

*/

## Input paramters

folder to scan:
`-f` `--folder` /path/to/code

recursively scan folder
`-r`

scan only one file
`--file` /path/to/code/file

results goes to, if provided
`-o` /path/to/output/folder/file

if `-o` was not provided just current {timestamp}_{root_folder_was_scanned}.md

## Output file

task content | /path/to/file

## Tech

Golang

## Build & usage

```sh
go build -o todoscanner .

./todoscanner -f ./src -r                 # scan recursively, report to ./{timestamp}_src.md
./todoscanner --folder ./src -o out/      # top-level files only, report into out/
./todoscanner --file ./src/main.go -o todos.md
```

Notes:

- `TODO` must be uppercase; the colon is optional (`// TODO fix` works too).
- A `//TODO` ends at the first non-`//` line, an empty `//` line, or another TODO.
  A `/* ... */` comment directly after it is appended to the TODO.
- Trailing comments (`x := 1 // TODO: ...`) and `/** TODO ... */` are recognised.
- Binary files and `.git`, `.hg`, `.svn`, `node_modules`, `vendor` folders are skipped.
- Each report row is `task | /abs/path/to/file:line`.

Run tests with `go test ./...`.
