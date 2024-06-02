package ignore

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
