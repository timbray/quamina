package quamina

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type valType int

const (
	stringType valType = iota
	numberType
	literalType
	existsTrueType
	existsFalseType
	shellStyleType
)

type typedVal struct {
	vType valType
	val   string
}
type patternField struct {
	path string
	vals []typedVal
}

type patternBuild struct {
	jd         *json.Decoder
	path       []string
	results    []*patternField
	isNameUsed map[string]bool
}

// patternFromJSON - I love naked returns and I cannot lie
func patternFromJSON(jsonBytes []byte) (fields []*patternField, namesUsed map[string]bool, err error) {
	// we can't use json.Unmarshal because it round-trips numbers through float64 and %f so they won't end up matching
	//  what the caller actually wrote in the patternField. json.Decoder is kind of slow due to excessive
	//  memory allocation, but I haven't got around to prematurely optimizing the addPattern code path
	var pb patternBuild
	pb.jd = json.NewDecoder(bytes.NewReader(jsonBytes))
	pb.isNameUsed = make(map[string]bool)
	pb.jd.UseNumber()
	t, err := pb.jd.Token()
	if err == io.EOF {
		err = errors.New("empty patternField")
		return
	} else if err != nil {
		err = errors.New("patternField is not a JSON object" + err.Error())
		return
	}
	switch tt := t.(type) {
	case json.Delim:
		if tt != '{' {
			err = errors.New("patternField is not a JSON object")
			return
		}
	default:
		err = errors.New("event is not a JSON object: doesn't start with '{'")
		return
	}

	err = readPatternObject(&pb)
	namesUsed = pb.isNameUsed
	fields = pb.results
	return
}

func readPatternObject(pb *patternBuild) error {
	for {
		t, err := pb.jd.Token()
		if err == io.EOF {
			return errors.New("event atEnd mid-object")
		} else if err != nil {
			return errors.New("pattern malformed: " + err.Error())
		}

		switch tt := t.(type) {
		case string:
			pb.isNameUsed[tt] = true
			pb.path = append(pb.path, tt)
			err = readPatternMember(pb)
			if err != nil {
				return err
			}
			pb.path = pb.path[:len(pb.path)-1]

		case json.Delim:
			if tt == '}' {
				return nil
			} else {
				return errors.New(fmt.Sprintf("floating '%v' in object", tt))
			}
		}
	}
}

func readPatternMember(pb *patternBuild) error {
	t, err := pb.jd.Token()
	if err == io.EOF {
		return errors.New("patternField atEnd mid-field")
	} else if err != nil {
		return errors.New("pattern malformed: " + err.Error())
	}

	switch tt := t.(type) {
	case json.Delim:
		switch tt {
		case '[':
			return readPatternArray(pb)
		case '{':
			return readPatternObject(pb)
		default:
			return errors.New(fmt.Sprintf("pattern malformed, illegal %v", tt))
		}
	default:
		return errors.New(fmt.Sprintf("pattern malformed, illegal %v", tt))
	}
}

func readPatternArray(pb *patternBuild) error {
	pathName := strings.Join(pb.path, "\n")
	var containsExclusive string
	elementCount := 0
	var pathVals []typedVal
	for {
		t, err := pb.jd.Token()
		if err == io.EOF {
			return errors.New("patternField atEnd mid-field")
		} else if err != nil {
			return errors.New("pattern malformed: " + err.Error())
		}

		switch tt := t.(type) {
		case json.Delim:
			if tt == ']' {
				if (containsExclusive != "") && (elementCount > 1) {
					return errors.New(fmt.Sprintf(`%s cannot be combined with other values in pattern`, containsExclusive))
				}
				pb.results = append(pb.results, &patternField{path: pathName, vals: pathVals})
				return nil
			} else if tt == '{' {
				var ce string
				pathVals, ce, err = readSpecialPattern(pb, pathVals)
				if ce != "" {
					containsExclusive = ce
				}
				if err != nil {
					return err
				}
			} else {
				return errors.New(fmt.Sprintf("pattern malformed, illegal %v", tt))
			}
		case string:
			pathVals = append(pathVals, typedVal{stringType, `"` + tt + `"`})
		case json.Number:
			pathVals = append(pathVals, typedVal{numberType, tt.String()})
		case bool:
			if tt {
				pathVals = append(pathVals, typedVal{literalType, "true"})
			} else {
				pathVals = append(pathVals, typedVal{literalType, "false"})
			}
		case nil:
			pathVals = append(pathVals, typedVal{literalType, "null"})
		}
		elementCount++
	}
}

func readSpecialPattern(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, containsExclusive string, err error) {
	containsExclusive = ""
	pathVals = valsIn
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	switch tt := t.(type) {
	case string:
		switch tt {
		case "exists":
			containsExclusive = tt
			pathVals, err = readExistsSpecial(pb, pathVals)
		case "shellstyle":
			pathVals, err = readShellStyleSpecial(pb, pathVals)
		default:
			err = errors.New("unrecognized in special pattern: " + tt)
		}
	default:
		err = errors.New("error reading name of special pattern")
	}
	return
}

func readExistsSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	pathVals = valsIn
	switch tt := t.(type) {
	case bool:
		if tt {
			pathVals = append(pathVals, typedVal{vType: existsTrueType})
		} else {
			pathVals = append(pathVals, typedVal{vType: existsFalseType})
		}
	default:
		err = errors.New("value for 'exists' pattern must be true or false")
		return
	}

	t, err = pb.jd.Token()
	if err != nil {
		return
	}
	switch tt := t.(type) {
	case json.Delim:
		if tt != '}' {
			err = errors.New(fmt.Sprintf("invalid character %v in 'existsMatches' pattern", tt))
		}
	default:
		err = errors.New("trailing garbage in 'existsMatches' pattern")
	}
	return
}
