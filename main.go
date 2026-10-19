package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// skipDirs are never descended into when scanning recursively.
var skipDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
	"vendor":       true,
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fset := flag.NewFlagSet("todoscanner", flag.ContinueOnError)
	var folder, file, output string
	var recursive bool
	fset.StringVar(&folder, "f", "", "folder to scan")
	fset.StringVar(&folder, "folder", "", "folder to scan (same as -f)")
	fset.BoolVar(&recursive, "r", false, "scan folder recursively")
	fset.StringVar(&file, "file", "", "scan only one file")
	fset.StringVar(&output, "o", "", "output .md file (default: ./{timestamp}_{scanned_folder}.md)")
	fset.Usage = func() {
		fmt.Fprintln(fset.Output(), "Usage: todoscanner (-f <folder> [-r] | --file <file>) [-o <output.md>]")
		fset.PrintDefaults()
	}
	if err := fset.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if (folder == "") == (file == "") {
		fset.Usage()
		return errors.New("exactly one of -f/--folder or --file must be provided")
	}

	now := time.Now()
	var (
		root  string
		files []string
		err   error
	)
	if file != "" {
		if root, err = filepath.Abs(file); err != nil {
			return err
		}
		info, err := os.Stat(root)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory, use -f instead", file)
		}
		files = []string{root}
	} else {
		if root, err = filepath.Abs(folder); err != nil {
			return err
		}
		if files, err = collectFiles(root, recursive); err != nil {
			return err
		}
	}

	outPath, err := resolveOutput(output, root, file != "", now)
	if err != nil {
		return err
	}

	var todos []Todo
	for _, path := range files {
		if path == outPath {
			continue
		}
		found, err := scanFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", path, err)
		}
		todos = append(todos, found...)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, []byte(renderMarkdown(todos, root, now)), 0o644); err != nil {
		return err
	}
	fmt.Printf("Found %d TODO(s) in %d file(s) scanned. Report: %s\n", len(todos), len(files), outPath)
	return nil
}

// collectFiles lists files in root, descending into subfolders only if recursive.
func collectFiles(root string, recursive bool) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory, use --file instead", root)
	}

	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", path, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if !recursive || skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// scanFile parses one file, skipping binary content.
func scanFile(path string) ([]Todo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	head := data
	if len(head) > 8000 {
		head = head[:8000]
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return nil, nil // binary file
	}
	return ParseTodos(bytes.NewReader(data), path)
}

// resolveOutput returns the absolute path of the report file. If output is
// empty, or points to an existing folder, the default name
// {timestamp}_{root_folder}.md is used.
func resolveOutput(output, root string, single bool, now time.Time) (string, error) {
	folderName := root
	if single {
		folderName = filepath.Dir(root)
	}
	name := filepath.Base(folderName)
	if name == string(filepath.Separator) || name == "." || name == "" {
		name = "root"
	}
	defaultName := fmt.Sprintf("%s_%s.md", now.Format("20060102_150405"), name)

	if output == "" {
		return filepath.Abs(defaultName)
	}
	if info, err := os.Stat(output); (err == nil && info.IsDir()) || strings.HasSuffix(output, string(filepath.Separator)) {
		output = filepath.Join(output, defaultName)
	}
	return filepath.Abs(output)
}

func renderMarkdown(todos []Todo, root string, now time.Time) string {
	var b strings.Builder
	b.WriteString("# TODOs\n\n")
	fmt.Fprintf(&b, "- Scanned: `%s`\n", root)
	fmt.Fprintf(&b, "- Generated: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- Total: %d\n\n", len(todos))

	if len(todos) == 0 {
		b.WriteString("No TODOs found.\n")
		return b.String()
	}

	b.WriteString("| Task | File |\n")
	b.WriteString("|------|------|\n")
	for _, t := range todos {
		text := t.Text
		if text == "" {
			text = "_(empty)_"
		}
		fmt.Fprintf(&b, "| %s | %s:%d |\n", escapeCell(text), escapeCell(t.File), t.Line)
	}
	return b.String()
}

func escapeCell(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}
