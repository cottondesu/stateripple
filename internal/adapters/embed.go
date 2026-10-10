package adapters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

func formerEmbeds(ctx context.Context, root, base string) (map[string][]string, error) {
	if base == "" {
		base = "HEAD^"
	}
	revData, err := command(ctx, root, "git", "rev-parse", "--verify", "--end-of-options", base+"^{commit}")
	if err != nil {
		return nil, err
	}
	rev := strings.TrimSpace(string(revData))
	data, err := command(ctx, root, "git", "ls-tree", "-r", "-z", "--full-tree", rev)
	if err != nil {
		return nil, err
	}
	sources, err := baseSources(data)
	if err != nil {
		return nil, err
	}
	result := map[string][]string{}
	var failures []error
	for _, file := range sources {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(append(failures, err)...)
		}
		source, err := command(ctx, root, "git", "cat-file", "blob", file.oid)
		if err != nil {
			failures = append(failures, fmt.Errorf("read base Go source %q (%s): %w", file.path, file.oid, err))
			continue
		}
		if !bytes.Contains(source, []byte("//go:embed")) {
			continue
		}
		patterns, err := sourceEmbeds(file.path, source)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		result[file.path] = patterns
	}
	return result, errors.Join(failures...)
}

func baseSources(data []byte) ([]headFile, error) {
	if len(data) > 0 && data[len(data)-1] != 0 {
		return nil, fmt.Errorf("unterminated base tree")
	}
	sources := []headFile{}
	for _, record := range strings.Split(string(data), "\x00") {
		if record == "" {
			continue
		}
		header, name, ok := strings.Cut(record, "\t")
		fields := strings.Fields(header)
		if !ok || name == "" || len(fields) != 3 {
			return nil, fmt.Errorf("invalid base tree record")
		}
		if fields[0] == "120000" && strings.HasSuffix(name, ".go") {
			return nil, fmt.Errorf("unsupported symbolic base Go source %q", name)
		}
		if fields[0] == "160000" || fields[0] == "120000" {
			continue
		}
		if fields[2] == "" || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
			return nil, fmt.Errorf("unsupported base tree entry %q", name)
		}
		if strings.HasSuffix(name, ".go") {
			sources = append(sources, headFile{name, fields[0], fields[2]})
		}
	}
	return sources, nil
}

func sourceEmbeds(name string, source []byte) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), name, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse base embed source %q: %w", name, err)
	}
	patterns := []string{}
	for _, group := range parsed.Comments {
		for _, comment := range group.List {
			text, matched := strings.CutPrefix(comment.Text, "//go:embed")
			if matched && len(text) > 0 && (text[0] == ' ' || text[0] == '\t') {
				values, err := embedPatterns(text)
				if err != nil {
					return nil, err
				}
				patterns = append(patterns, values...)
			}
		}
	}
	return patterns, nil
}

func embedPatterns(text string) ([]string, error) {
	patterns := []string{}
	for text = strings.TrimSpace(text); text != ""; text = strings.TrimSpace(text) {
		var value string
		if text[0] == '"' || text[0] == '`' {
			quote := text[0]
			end := 1
			for end < len(text) {
				if text[end] == quote {
					break
				}
				if quote == '"' && text[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(text) {
				return nil, fmt.Errorf("invalid base embed pattern")
			}
			decoded, err := strconv.Unquote(text[:end+1])
			if err != nil {
				return nil, fmt.Errorf("base embed pattern: %w", err)
			}
			value = decoded
			text = text[end+1:]
		} else {
			end := strings.IndexFunc(text, unicode.IsSpace)
			if end < 0 {
				end = len(text)
			}
			value = text[:end]
			text = text[end:]
		}
		patterns = append(patterns, value)
	}
	return patterns, nil
}

func formerEmbedOwners(root string, packages []goPackage, patterns map[string][]string, changes []string) map[string][]owner {
	dirs := map[string]goPackage{}
	for _, p := range packages {
		rel, err := filepath.Rel(root, p.Dir)
		if err == nil {
			dirs[filepath.ToSlash(rel)] = p
		}
	}
	result := map[string][]owner{}
	for source, patterns := range patterns {
		dir := path.Dir(source)
		p, known := dirs[dir]
		target := owner{classification: "former_embed", risk: 3}
		if known {
			target.id = "pkg:" + p.ImportPath
		}
		if strings.HasSuffix(source, "_test.go") {
			target.classification = "former_test_embed"
			target.risk = 2
			target.id = ""
			if known && hasTests(p) {
				target.id = "test:" + p.ImportPath
			}
		}
		for _, changed := range changes {
			relative, err := filepath.Rel(filepath.FromSlash(dir), filepath.FromSlash(changed))
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				continue
			}
			relative = filepath.ToSlash(relative)
			for _, pattern := range patterns {
				pattern = strings.TrimPrefix(pattern, "all:")
				matched := false
				for candidate := relative; candidate != "."; candidate = path.Dir(candidate) {
					if ok, err := path.Match(pattern, candidate); err != nil || ok {
						matched = true
						break
					}
				}
				if matched {
					result[changed] = append(result[changed], target)
					break
				}
			}
		}
	}
	return result
}
