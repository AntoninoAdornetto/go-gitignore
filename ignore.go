package ignore

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
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
	return iPattern
}
