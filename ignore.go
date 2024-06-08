package ignore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	globstar "github.com/bmatcuk/doublestar/v4"
)

type Bits = uint8

const (
	FLAG_NO_DIR Bits = 1 << iota
	FLAG_MUST_BE_DIR
	FLAG_WILDCARD
	FLAG_MATCHER
	FLAG_RANGE_NOTATION
	FLAG_NEGATE
)

type Ignorer struct {
	ExcludeGroups []ExcludeGroup
}

type ExcludeGroup struct {
	Src         string
	BasePath    string
	RecordCount int
	PatternList []IgnorePattern
}

type IgnorePattern struct {
	Flags           uint8
	Pattern         string
	OriginalPattern string
}

func NewIgnorer(absPath string) (*Ignorer, error) {
	ig := &Ignorer{}

	if err := ig.AppendExcludeGroup(absPath, ".gitignore"); err != nil {
		return nil, err
	}

	if err := ig.AppendExcludeGroup(absPath, ".git/info/exclude"); err != nil {
		if !os.IsNotExist(err) {
			return nil, nil
		}
	}

	return ig, nil
}

func NewExcludeGroup(basePath, filePath string) (*ExcludeGroup, error) {
	group := &ExcludeGroup{Src: filePath, BasePath: filepath.Clean(basePath)}

	iFile, err := os.Open(filepath.Join(basePath, filePath))
	if err != nil {
		return nil, err
	}

	defer iFile.Close()
	iPatterns, err := ScanPatterns(iFile)
	if err != nil {
		return nil, err
	}

	group.RecordCount = len(iPatterns)
	group.PatternList = iPatterns
	return group, nil
}

func (ig *Ignorer) AppendExcludeGroup(path, src string) error {
	group, err := NewExcludeGroup(path, src)
	if err != nil {
		return err
	}

	ig.ExcludeGroups = append(ig.ExcludeGroups, *group)
	return nil
}

func ScanPatterns(r io.Reader) ([]IgnorePattern, error) {
	iPatterns := make([]IgnorePattern, 0, 10)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		line = bytes.TrimFunc(line, unicode.IsSpace)
		iPattern := NewIgnorePattern(line)
		iPatterns = append(iPatterns, iPattern)
	}

	return iPatterns, scanner.Err()
}

func NewIgnorePattern(line []byte) IgnorePattern {
	return parsePattern(line)
}

func parsePattern(line []byte) IgnorePattern {
	iPattern := IgnorePattern{OriginalPattern: string(line)}
	builder := strings.Builder{}
	separatorCount := 0

	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '!':
			iPattern.onNegateCase(&builder, i)
		case '/':
			separatorCount++
			iPattern.onSeparatorCase(&builder, i, line)
		case '*':
			iPattern.onWildcardCase(&builder)
		case '?':
			iPattern.onCharMatcherCase(&builder)
		case '[':
			i = iPattern.onRangeCase(&builder, i, line)
		default:
			builder.WriteByte(line[i])
		}
	}

	if separatorCount == 0 {
		iPattern.Flags |= FLAG_NO_DIR
	}

	iPattern.Pattern = builder.String()
	return iPattern
}

func (iPat *IgnorePattern) onNegateCase(builder *strings.Builder, i int) {
	if i == 0 {
		iPat.Flags |= FLAG_NEGATE
		return
	}

	builder.WriteByte('!')
}

func (iPat *IgnorePattern) onSeparatorCase(builder *strings.Builder, i int, line []byte) {
	if i == 0 {
		// @TODO skipping out on writing the leading dir separator is a temp fix, revisit.
		return
	}

	if i == len(line)-1 {
		iPat.Flags |= FLAG_MUST_BE_DIR
		return
	}

	builder.WriteByte('/')
}

func (iPat *IgnorePattern) onWildcardCase(builder *strings.Builder) {
	builder.WriteByte('*')
	iPat.Flags |= FLAG_WILDCARD
}

func (iPat *IgnorePattern) onCharMatcherCase(builder *strings.Builder) {
	iPat.Flags |= FLAG_MATCHER
	builder.WriteByte('?')
}

// @TODO check into nested range notation, is that possible? May need to utilize a stack based approach
func (iPat *IgnorePattern) onRangeCase(builder *strings.Builder, i int, line []byte) int {
	start, end := i, i
	balanced := false

	for ; start < len(line); end++ {
		if line[end] == ']' {
			balanced = true
			break
		}
	}

	if !balanced {
		panic(fmt.Sprintf("unbalanced range notation pattern: %s", string(line)))
	}

	builder.Write(line[start : end+1])
	iPat.Flags |= FLAG_RANGE_NOTATION
	return end
}

func (iPat *IgnorePattern) hasFlag(flag Bits) bool {
	return iPat.Flags&flag != 0
}

func (ig *Ignorer) Match(path string) (bool, error) {
	for _, group := range ig.ExcludeGroups {
		if match, err := group.Match(path); err != nil || match {
			return match, err
		}
	}
	return false, nil
}

func (group *ExcludeGroup) Match(path string) (bool, error) {
	rel, err := filepath.Rel(group.BasePath, path)
	if err != nil {
		return false, err
	}

	for _, p := range group.PatternList {
		matched, err := p.Match(rel)
		if err != nil {
			return false, err
		}

		if matched {
			if p.hasFlag(FLAG_NEGATE) {
				return false, nil
			}
			return true, nil
		}
	}

	return false, nil
}

func (iPat *IgnorePattern) Match(path string) (bool, error) {
	last := filepath.Base(path)
	switch {
	case iPat.hasFlag(FLAG_NO_DIR):
		return filepath.Match(iPat.Pattern, last)
	case iPat.hasFlag(FLAG_WILDCARD):
		return globstar.Match(iPat.Pattern, path)
	default:
		return filepath.Match(iPat.Pattern, path)
	}
}
