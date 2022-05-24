package quamina

import "path/filepath"

var testDataDir = "../../test_data"

func testData(filename string) string {
	return filepath.Join(testDataDir, filename)
}
