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

	ignorePath := filepath.Join(absPath, ".gitignore")
	excludePath := filepath.Join(absPath, ".git", "info", "exclude")

	if err := ig.AppendExcludeGroup(ignorePath, ".gitignore"); err != nil {
		return nil, err
	}

	if err := ig.AppendExcludeGroup(excludePath, "exclude"); err != nil {
		return nil, err
	}

	return ig, nil
}

func NewExcludeGroup(path, src string) (*ExcludeGroup, error) {
	group := &ExcludeGroup{Src: src, BasePath: filepath.Clean(path)}

	iFile, err := os.Open(path)
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
			i = iPattern.onWildcardCase(&builder, i, line)
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
	if i == len(line)-1 {
		iPat.Flags |= FLAG_MUST_BE_DIR
		return
	}

	builder.WriteByte('/')
}

func (iPat *IgnorePattern) onWildcardCase(builder *strings.Builder, i int, line []byte) int {
	increment := i
	if i+1 < len(line) && line[i+1] == '*' {
		increment++
	}

	builder.WriteByte('*')
	iPat.Flags |= FLAG_WILDCARD
	return increment
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
