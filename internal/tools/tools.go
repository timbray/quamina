package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// TestDataDir is the default relative pathname for test data.
//
// Test code can update this variable (so we don't have to try to
// search).  Most tests are run from pkg/*/, so this default is
// reasonable.
var TestDataDir = "../test_data"

// Exists reports whether the given filename exists.
func Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// TestDataFilename returns TestDataDir/basename if that file exists;
// otherwise the function returns an error.
func TestDataFilename(basename string) (string, error) {
	candidate := filepath.Join(TestDataDir, basename)
	if exists, err := Exists(candidate); err != nil {
		return "", nil
	} else if !exists {
		return "", os.ErrNotExist
	}

	return candidate, nil
}

// MustTestDataFilename calls TestDataFilename and panics if the
// filename doesn't exist.
func MustTestDataFilename(basename string) string {
	filename, err := TestDataFilename(basename)
	if err == nil {
		return filename
	}
	panic(fmt.Errorf("didn't file test data for '%s'", basename))
}
