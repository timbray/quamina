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
	anythingButType
	prefixType
	monocaseType
	wildcardType
	regexpType
)

// typedVal represents the value of a field in a pattern, giving the value and the type of pattern.
// - list is used to handle anything-but matches with multiple values.
// - parsedRegexp only used for vType == regexpType
type typedVal struct {
	vType        valType
	val          string
	list         [][]byte
	parsedRegexp regexpRoot
}

// patternField represents a field in a pattern.
// vals is a list because field values are always given as a JSON array.
type patternField struct {
	path string
	vals []typedVal
}

// patternBuild tracks the progress of patternFromJSON through a pattern-compilation project.
type patternBuild struct {
	jd      *json.Decoder
	path    []string
	results []*patternField
}

// patternFromJSON compiles a JSON text provided in jsonBytes into a list of patternField structures.
// I love naked returns and I cannot lie
func patternFromJSON(jsonBytes []byte) (fields []*patternField, err error) {
	// we can't use json.Unmarshal because it round-trips numbers through float64 and %f, so they won't end up matching
	// what the caller actually wrote in the patternField. json.Decoder is kind of slow due to excessive
	// memory allocation, but I haven't got around to prematurely optimizing the patternFromJSON code path
	var pb patternBuild
	pb.jd = json.NewDecoder(bytes.NewReader(jsonBytes))
	pb.jd.UseNumber()

	// we use the tokenizer rather than pulling the pattern in with UnMarshall
	t, err := pb.jd.Token()
	if errors.Is(err, io.EOF) {
		err = errors.New("empty Pattern")
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
	fields = pb.results
	return
}

func readPatternObject(pb *patternBuild) error {
	for {
		t, err := pb.jd.Token()
		if errors.Is(err, io.EOF) {
			return errors.New("event atEnd mid-object")
		} else if err != nil {
			return errors.New("pattern malformed: " + err.Error())
		}

		switch tt := t.(type) {
		case string:
			pb.path = append(pb.path, tt)
			err = readPatternMember(pb)
			if err != nil {
				return err
			}
			pb.path = pb.path[:len(pb.path)-1]

		case json.Delim:
			// has to be '}' or the tokenizer would have thrown an error
			return nil
		}
	}
}

func readPatternMember(pb *patternBuild) error {
	t, err := pb.jd.Token()
	if errors.Is(err, io.EOF) {
		return errors.New("pattern ends mid-field")
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
		default: // can't happen
			return fmt.Errorf("pattern malformed, illegal %v", tt)
		}
	default:
		return fmt.Errorf("pattern malformed, illegal %v", tt)
	}
}

func readPatternArray(pb *patternBuild) error {
	pathName := strings.Join(pb.path, SegmentSeparator)
	var containsExclusive string
	elementCount := 0
	var pathVals []typedVal
	for {
		t, err := pb.jd.Token()
		if errors.Is(err, io.EOF) {
			return errors.New("patternField atEnd mid-field")
		} else if err != nil {
			// can't happen
			return errors.New("pattern malformed: " + err.Error())
		}

		switch tt := t.(type) {
		case json.Delim:
			if tt == ']' {
				if (containsExclusive != "") && (elementCount > 1) {
					return fmt.Errorf(`%s cannot be combined with other values in pattern`, containsExclusive)
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
				return fmt.Errorf("pattern malformed, illegal %v", tt)
			}
		case string:
			pathVals = append(pathVals, typedVal{vType: stringType, val: `"` + tt + `"`})
		case json.Number:
			pathVals = append(pathVals, typedVal{vType: numberType, val: tt.String()})
		case bool:
			if tt {
				pathVals = append(pathVals, typedVal{vType: literalType, val: "true"})
			} else {
				pathVals = append(pathVals, typedVal{vType: literalType, val: "false"})
			}
		case nil:
			pathVals = append(pathVals, typedVal{vType: literalType, val: "null"})
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

	// tokenizer will throw an error if it's not a string
	tt := t.(string)
	switch tt {
	case "anything-but":
		containsExclusive = tt
		pathVals, err = readAnythingButSpecial(pb, pathVals)
	case "exists":
		containsExclusive = tt
		pathVals, err = readExistsSpecial(pb, pathVals)
	case "shellstyle":
		pathVals, err = readShellStyleSpecial(pb, pathVals)
	case "wildcard":
		pathVals, err = readWildcardSpecial(pb, pathVals)
	case "prefix":
		pathVals, err = readPrefixSpecial(pb, pathVals)
	case "equals-ignore-case":
		pathVals, err = readMonocaseSpecial(pb, pathVals)
	case "regexp":
		containsExclusive = tt
		pathVals, err = readRegexpSpecial(pb, pathVals)
	default:
		err = errors.New("unrecognized in special pattern: " + tt)
	}
	return
}

func readPrefixSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	pathVals = valsIn

	prefixString, ok := t.(string)
	if !ok {
		err = errors.New("value for 'prefix' must be a string")
		return
	}
	val := typedVal{
		vType: prefixType,
		val:   `"` + prefixString + `"`,
	}
	pathVals = append(pathVals, val)

	// has to be } or tokenizer will throw error
	_, err = pb.jd.Token()
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
	switch t.(type) {
	case json.Delim:
		// no-op, has to be }
	default:
		err = errors.New("trailing garbage in 'existsMatches' pattern")
	}
	return
}
