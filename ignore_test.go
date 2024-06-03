package ignore_test

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	ignore "github.com/AntoninoAdornetto/go-gitignore"
)

/*
testdata/.gitignore - contains the patterns used for test cases
testdata/results.json - contains the expected results we should get after parsing each pattern

If additional tests need to be added, see the bit flags in ignore.go for how to calculate the
decimal value. It's pretty straight forward but wanted to mention it just in case
*/

func TestScanPatterns(t *testing.T) {
	ig := ignore.Ignorer{}
	err := ig.AppendExcludeGroup("./testdata/.gitignore", ".gitignore")
	assertExcludeGroup(t, &ig.ExcludeGroups[0], err)

	expectedResults := readResultData(t)
	actualResults := ig.ExcludeGroups[0]

	for i, expected := range expectedResults.Results {
		original := expected.Original
		formatted := expected.Formatted
		flags := expected.Flags
		actual := actualResults.PatternList[i]

		if original != actual.OriginalPattern {
			t.Fatalf(
				"expected original pattern to be %s but got %s",
				original,
				actual.OriginalPattern,
			)
		}

		if formatted != actual.Pattern {
			t.Fatalf("expected formatted pattern to be %s but got %s", formatted, actual.Pattern)
		}

		// heads up, the flags in json have to be converted from binary to decmial.
		if flags != actual.Flags {
			t.Fatalf("expected flags for %s to be %d but got %d", formatted, flags, actual.Flags)
		}
	}
}

func TestNewIgnorer(t *testing.T) {
	ig, err := ignore.NewIgnorer("./")
	assertIgnorer(t, ig, err)

	// creates 2 exclude groups. 1 for the root .gitignore file and 1 for the .git/info/exclude file
	groups := ig.ExcludeGroups
	if len(groups) != 2 {
		t.Fatalf("expected 2 exclude groups but got %d", len(groups))
	}

	// attempt to create a new ignorer with a bad path
	_, err = ignore.NewIgnorer("./testdata/unknowndir")
	if err == nil {
		t.Log("expected an error but got nil")
		t.FailNow()
	}
}

func TestNewExcludeGroup(t *testing.T) {
	// .gitignore exclude group
	group, err := ignore.NewExcludeGroup("./.gitignore", ".gitignore")
	assertExcludeGroup(t, group, err)

	// .git/info/exclude exclude group
	group, err = ignore.NewExcludeGroup("./.git/info/exclude", "exclude")
	assertExcludeGroup(t, group, err)

	// error when path does not contain a .gitignore or exclude file
	group, err = ignore.NewExcludeGroup("./testdata/unknownpath", ".gitignore")
	if err == nil {
		t.Log("expected to receive an error but got nil")
		t.FailNow()
	}

	if group != nil {
		t.Log("expected exclude group to be nil")
		t.FailNow()
	}

	group, err = ignore.NewExcludeGroup("./testdata/unknownpath", "exclude")
	if err == nil {
		t.Log("expected to receive an error but got nil")
		t.FailNow()
	}

	if group != nil {
		t.Log("expected exclude group to be nil")
		t.FailNow()
	}
}

func TestAppendExcludeGroup(t *testing.T) {
	ig := ignore.Ignorer{}

	err := ig.AppendExcludeGroup("./.gitignore", ".gitignore")
	if err != nil {
		t.Fatalf("expected append error to be nil but got %s", err.Error())
	}

	if len(ig.ExcludeGroups) != 1 {
		t.Fatalf("expected exclude groups to have length of 1 but got %d", len(ig.ExcludeGroups))
	}

	err = ig.AppendExcludeGroup("./.git/info/exclude", "exclude")
	if err != nil {
		t.Fatalf("expected append error to be nil but got %s", err.Error())
	}

	if len(ig.ExcludeGroups) != 2 {
		t.Fatalf("expected exclude groups to have length of 2 but got %d", len(ig.ExcludeGroups))
	}

	err = ig.AppendExcludeGroup("./unknown.gitignore", ".gitignore")
	if err == nil {
		t.Log("expected an err but got nil")
		t.FailNow()
	}

	if len(ig.ExcludeGroups) != 2 {
		t.Fatalf("expected exclude groups to have length of 2 but got %d", len(ig.ExcludeGroups))
	}
}

func assertIgnorer(t *testing.T, ig *ignore.Ignorer, err error) {
	if err != nil {
		t.Fatalf("expected ignorer instantiation to not return and error but got %s", err.Error())
	}

	if ig == nil {
		t.Log("expected ignorer struct not to be nill")
		t.FailNow()
	}
}

func assertExcludeGroup(t *testing.T, exc *ignore.ExcludeGroup, err error) {
	if err != nil {
		t.Fatalf("expected exclude group instantiation to not return an error but got %s", err)
	}

	if exc == nil {
		t.Log("expected exclude group not to be nill")
		t.FailNow()
	}
}

type results struct {
	Results []resultList `json:"results"`
}

type resultList struct {
	Original  string `json:"original-pattern"`
	Formatted string `json:"formatted-pattern"`
	Flags     uint8  `json:"flags"`
}

func readResultData(t *testing.T) results {
	f, err := os.Open("./testdata/results.json")
	if err != nil {
		t.Fatalf("expected to not receive an error but got %s", err.Error())
	}

	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("expected to not receive an error but got %s", err.Error())
	}

	results := results{}
	err = json.Unmarshal(data, &results)
	if err != nil {
		t.Fatalf("expected unmarshal error to be nil but got %s", err.Error())
	}

	return results
}
