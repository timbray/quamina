package quamina

import "testing"

func TestAddX(t *testing.T) {
	set := newMatchSet()

	// empty exes
	set = set.addX()
	if !isSameMatches(set) {
		t.Errorf("Expected matches to be empty: %+v", set.matches())
	}

	newSet := set.addX(1)
	// existing set should be empty.
	if len(set.matches()) > 0 {
		t.Errorf("Expected matches to be empty: %+v", set.matches())
	}
	if !isSameMatches(newSet, 1) {
		t.Errorf("Expected matches to be [1]: %+v", set.matches())
	}

	// add another two values
	newSet = newSet.addX(2)
	newSet = newSet.addX(3)
	if !isSameMatches(newSet, 1, 2, 3) {
		t.Errorf("Expected matches to be [1, 2, 3]: %+v", set.matches())
	}
}

func TestAddXSingleThreaded(t *testing.T) {
	set := newMatchSet()

	// empty exes
	set.addXSingleThreaded()
	if !isSameMatches(set) {
		t.Errorf("Expected matches to be empty: %+v", set.matches())
	}

	set.addXSingleThreaded(1)
	// existing set should be empty.
	if !isSameMatches(set, 1) {
		t.Errorf("Expected matches to be [1]: %+v", set.matches())
	}

	// add another two values
	set.addXSingleThreaded(2)
	set.addXSingleThreaded(3)
	if !isSameMatches(set, 1, 2, 3) {
		t.Errorf("Expected matches to be [1, 2, 3]: %+v", set.matches())
	}
}

func isSameMatches(matchSet *matchSet, exes ...X) bool {
	if len(exes) == 0 && len(matchSet.matches()) == 0 {
		return true
	}

	if len(exes) != len(matchSet.matches()) {
		return false
	}

	for _, x := range exes {
		if !matchSet.contains(x) {
			return false
		}
	}

	return true
}
