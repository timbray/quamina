package tools

import (
	"testing"
)

func TestFindTestData(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		filename, err := TestDataFilename("cl-sample-0")
		if err != nil {
			t.Fatal(err)
		}
		if filename == "" {
			t.Fatal(filename)
		}
	})
	t.Run("notfound", func(t *testing.T) {
		_, err := TestDataFilename("queso-0")
		if err == nil {
			t.Fatal("expected protest")
		}
	})
}
