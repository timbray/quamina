package quamina

import (
	"fmt"
	"strconv"
	"unicode/utf16"
)

// flattenJSON is a custom non-general-purpose JSON parser whose object is to implement Flattener and produce a []Field
//  list from a JSON object.  This could be done (and originally was) with the built-in encoding/json, but the
//  performance was unsatisfactory (99% of time spent parsing events < 1% matching them). The profiler suggests
//  that the performance issue was mostly due to excessive memory allocation.
// If we assume that the event is immutable while we're working, then all the pieces of it that constitute
//  the fields & values can be represented as []byte slices using a couple of offsets into the underlying event.
//  There is an exception, namely strings that contain \-prefixed JSON escapes; since we want to work with the
//  actual UTF-8 bytes, this requires re-writing such strings into memory we have to allocate.
// TODO: There are gaps in the unit-test coverage, including nearly all the error conditions
type flattenJSON struct {
	event      []byte      // event being processed, treated as immutable
	eventIndex int         // current byte index into the event
	fields     []Field     // the under-construction return value of the Flatten method
	skipping   int         // track whether we're within the scope of a segment that isn't used
	arrayTrail []ArrayPos  // current array-position cookie crumbs
	arrayCount int32       // how many arrays we've seen, used in building arrayTrail
	tracker    NameTracker // knows if a segment is used; if not, no need to process
	cleanSheet bool        // initially true, don't have to call Reset()
}

// Reset an flattenJSON struct so it can be re-used and won't need to be reconstructed for each event to be flattened
func (fj *flattenJSON) reset() {
	fj.eventIndex = 0
	fj.fields = fj.fields[:0]
	fj.skipping = 0
	fj.arrayTrail = fj.arrayTrail[:0]
	fj.arrayCount = 0
}

// JSON literals
var trueBytes = []byte("true")

var (
	falseBytes = []byte("false")
	nullBytes  = []byte("null")
)

// fjState - this is a finite state machine parser, or rather a collection of smaller FSM parsers. Some of these
//  states are used in only one function, others in multiple places
type fjState int

const (
	startState fjState = iota
	inObjectState
	seekingColonState
	memberValueState
	inArrayState
	afterValueState
	numberStartState
	numberIntegralPartState
	numberFracState
	numberAfterEState
	numberExpState
	trailerState
	startEscapeState
	wantEscapeUState
	readHexDigitState
)

func newJSONFlattener() Flattener {
	return &flattenJSON{fields: make([]Field, 0, 32), cleanSheet: true}
}

func (fj *flattenJSON) Copy() Flattener {
	return newJSONFlattener()
}

// Flatten implements the Flattener interface. It assumes that the event is immutable - if you modify the event
//  bytes while the matcher is running, grave disorder will ensue.
func (fj *flattenJSON) Flatten(event []byte, tracker NameTracker) ([]Field, error) {
	if fj.cleanSheet {
		fj.cleanSheet = false
	} else {
		fj.reset()
	}
	if len(event) == 0 {
		return nil, fj.error("empty event")
	}
	var err error
	fj.event = event
	fj.tracker = tracker
	state := startState
	for {
		ch := fj.ch()
		switch state {
		case startState:
			switch ch {
			// single top-level object
			case '{':
				err = fj.readObject(nil)
				if err != nil {
					return nil, err
				}
				state = trailerState

			case ' ', '\t', '\n', '\r':
			// no-op
			default:
				return nil, fj.error("not a JSON object")
			}

		// eat trailing white space, if any
		case trailerState:
			switch ch {
			case ' ', '\t', '\n', '\r':
				// no-op
			default:
				return nil, fj.error(fmt.Sprintf("garbage char '%c' after top-level object", ch))
			}
		}

		// optimization to avoid calling step() and expensively construct an error object at the end of each event
		fj.eventIndex++
		if fj.eventIndex == len(fj.event) {
			return fj.fields, nil
		}
	}
}

// readObject - process through a JSON object, recursing if necessary into sub-objects
func (fj *flattenJSON) readObject(pathName []byte) error {
	var err error
	state := inObjectState

	// eventIndex points at {
	err = fj.step()
	if err != nil {
		return err
	}

	// make a snapshot of the current ArrayPos trail for use in any member fields, because it doesn't change in
	//  the course of reading an object
	var arrayTrail []ArrayPos
	if fj.skipping == 0 {
		arrayTrail = make([]ArrayPos, len(fj.arrayTrail))
		copy(arrayTrail, fj.arrayTrail)
	}

	// memberName contains the field-name we're processing
	var memberName []byte
	var memberIsUsed bool
	isLeaf := false
	for {
		ch := fj.ch()
		switch state {
		case inObjectState:
			switch ch {
			case ' ', '\t', '\n', '\r':
				// no-op
			case '"':
				memberName, err = fj.readMemberName()
				if err != nil {
					return err
				}
				memberIsUsed = (fj.skipping == 0) && fj.tracker.IsNameUsed(memberName)
				state = seekingColonState
			default:
				return fj.error(fmt.Sprintf("illegal character %c in JSON object", ch))
			}
		case seekingColonState:
			switch ch {
			case ' ', '\t', '\n', '\r':
				// no-op
			case ':':
				state = memberValueState
			default:
				return fj.error(fmt.Sprintf("illegal character %c while looking for colon", ch))
			}
		case memberValueState:
			// bypass space between colon and value. A bit klunky but allows for immense simplification
			// TODO: Investigate if there's a more efficient way to say this, or should just trust Go compiler
			for ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
				if fj.step() != nil {
					return fj.error("event truncated after colon")
				}
				ch = fj.ch()
			}

			var val []byte
			var alt []byte
			switch ch {
			case '"':
				val, err = fj.readStringValue()
				isLeaf = true
			case 't':
				val, err = fj.readLiteral(trueBytes)
				isLeaf = true
			case 'f':
				val, err = fj.readLiteral(falseBytes)
				isLeaf = true
			case 'n':
				val, err = fj.readLiteral(nullBytes)
				isLeaf = true
			case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				val, alt, err = fj.readNumber()
				isLeaf = true
			case '[':
				if !fj.tracker.IsNameUsed(memberName) {
					fj.skipping++
				}
				var arrayPath []byte
				if fj.skipping == 0 {
					arrayPath = pathForChild(pathName, memberName)
				}
				err = fj.readArray(arrayPath)
				if err != nil {
					return err
				}
				if !fj.tracker.IsNameUsed(memberName) {
					fj.skipping--
				}
			case '{':
				if !fj.tracker.IsNameUsed(memberName) {
					fj.skipping++
				}
				var objectPath []byte
				if fj.skipping == 0 {
					objectPath = pathForChild(pathName, memberName)
				}
				err = fj.readObject(objectPath)
				if err != nil {
					return err
				}
				if !fj.tracker.IsNameUsed(memberName) {
					fj.skipping--
				}
			default:
				return fj.error(fmt.Sprintf("illegal character %c after field name", ch))
			}
			if isLeaf {
				if err != nil {
					return err
				}
			}
			if val != nil {
				if memberIsUsed {
					fj.storeObjectMemberField(pathForChild(pathName, memberName), arrayTrail, val)
				}
			}
			if alt != nil {
				alt = nil
			}
			state = afterValueState
		case afterValueState:
			switch ch {
			case ',':
				state = inObjectState
			case '}':
				return nil
			case ' ', '\t', '\n', '\r':
				// no-op
			default:
				return fj.error(fmt.Sprintf("illegal character %c in object", ch))
			}
		}
		err = fj.step()
		if err != nil {
			return err
		}
	}
}

func (fj *flattenJSON) readArray(pathName []byte) error {
	// eventIndex points at [
	var err error
	err = fj.step()
	if err != nil {
		return err
	}
	// these maintain the arraytrail state
	if fj.skipping == 0 {
		fj.enterArray()
		defer fj.leaveArray()
	}

	state := inArrayState
	isLeaf := false
	for {
		ch := fj.ch()
		var val []byte // resets on each loop
		var alt []byte
		switch state {
		case inArrayState:
			// bypass space before element value. A bit klunky but allows for immense simplification
			for ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
				if fj.step() != nil {
					return fj.error("event truncated within array")
				}
				ch = fj.ch()
			}

			switch ch {
			case '"':
				val, err = fj.readStringValue()
				isLeaf = true
			case 't':
				val, err = fj.readLiteral(trueBytes)
				isLeaf = true
			case 'f':
				val, err = fj.readLiteral(falseBytes)
				isLeaf = true
			case 'n':
				val, err = fj.readLiteral(nullBytes)
				isLeaf = true
			case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				val, alt, err = fj.readNumber()
				isLeaf = true
			case '{':
				if fj.skipping == 0 {
					fj.stepOneArrayElement()
				}
				err = fj.readObject(pathName)
				if err != nil {
					return err
				}
			case '[':
				if fj.skipping == 0 {
					fj.stepOneArrayElement()
				}
				err = fj.readArray(pathName)
				if err != nil {
					return err
				}
			default:
				return fj.error(fmt.Sprintf("illegal character %c in array", ch))
			}
			if isLeaf {
				if err != nil {
					return err
				}
			}
			if val != nil {
				if fj.skipping == 0 {
					fj.stepOneArrayElement()
					fj.storeArrayElementField(pathName, val)
				}
			}
			if alt != nil {
				alt = nil
			}
			state = afterValueState
		case afterValueState:
			switch ch {
			case ']':
				return nil
			case ',':
				state = inArrayState
			case ' ', '\t', '\n', '\r':
				// no-op
			default:
				return fj.error(fmt.Sprintf("illegal character %c in array", ch))
			}
		}
		err = fj.step()
		if err != nil {
			return err
		}
	}
}

/*
 * Note that these functions that read leaf values often have to back up the eventIndex when they hit the character
 *  that signifies the end of what they're parsing, so that a higher-level matcher can evaluate it, because all
 *  these higher-level funcs are going to advance the pointer after each invocation
 */

func (fj *flattenJSON) readNumber() ([]byte, []byte, error) {
	// points at the first character in the number
	numStart := fj.eventIndex
	state := numberStartState
	for {
		ch := fj.ch()
		switch state {
		case numberStartState:
			switch ch {
			case '-':
				state = numberIntegralPartState
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				state = numberIntegralPartState
			}
		case numberIntegralPartState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			case '.':
				state = numberFracState
			case 'e', 'E':
				state = numberAfterEState
			case ',', ']', '}', ' ', '\t', '\n', '\r':
				fj.eventIndex--
				// TODO: Too expensive; make it possible for people to ask for this
				// bytes := fj.event[numStart : fj.eventIndex+1]
				// c, err := canonicalize(bytes)
				var alt []byte
				//if err == nil {
				//	alt = []byte(c)
				//}
				return fj.event[numStart : fj.eventIndex+1], alt, nil
			default:
				return nil, nil, fj.error(fmt.Sprintf("illegal char '%c' in number", ch))
			}
		case numberFracState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			case ',', ']', '}', ' ', '\t', '\n', '\r':
				fj.eventIndex--
				bytes := fj.event[numStart : fj.eventIndex+1]
				// TODO: Too expensive; make it possible for people to ask for this
				// c, err := canonicalize(bytes)
				var alt []byte
				//if err == nil {
				//	alt = []byte(c)
				//}
				return bytes, alt, nil
			case 'e', 'E':
				state = numberAfterEState
			default:
				return nil, nil, fj.error(fmt.Sprintf("illegal char '%c' in number", ch))
			}
		case numberAfterEState:
			switch ch {
			case '-', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			default:
				return nil, nil, fj.error(fmt.Sprintf("illegal char '%c' after 'e' in number", ch))
			}
			state = numberExpState

		case numberExpState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			case ',', ']', '}', ' ', '\t', '\n', '\r':
				fj.eventIndex--
				// bytes := fj.event[numStart : fj.eventIndex+1]
				// TODO: Too expensive; make it possible for people to ask for this
				// c, err := canonicalize(bytes)
				var alt []byte
				// if err == nil {
				//	alt = []byte(c)
				// }
				return fj.event[numStart : fj.eventIndex+1], alt, nil
			default:
				return nil, nil, fj.error(fmt.Sprintf("illegal char '%c' in exponent", ch))
			}
		}
		if fj.step() != nil {
			return nil, nil, fj.error("event truncated in number")
		}
	}
}

func (fj *flattenJSON) readLiteral(literal []byte) ([]byte, error) {
	for _, literalCh := range literal {
		if literalCh != fj.ch() {
			return nil, fj.error("unknown literal")
		}
		if fj.step() != nil {
			return nil, fj.error("truncated literal value")
		}
	}
	fj.eventIndex--
	return literal, nil
}

// we're positioned at the " that marks the start of a string value in an array or object.
//  ideally, we'd like to construct the member name as just a slice of the event buffer,
//  but will have to find a new home for it if it has JSON \-escapes
func (fj *flattenJSON) readStringValue() ([]byte, error) {
	// value includes leading and trailng "
	valStart := fj.eventIndex
	if fj.step() != nil {
		return nil, fj.error("event truncated in mid-string")
	}
	for {
		ch := fj.ch()
		if ch == '"' {
			return fj.event[valStart : fj.eventIndex+1], nil
		} else if ch == '\\' {
			val, err := fj.readStringValWithEscapes(valStart)
			return val, err
		} else if ch <= 0x1f || ch >= byte(byteCeiling) {
			return nil, fj.error(fmt.Sprintf("illegal UTF-8 byte %x in string value", ch))
		}
		if fj.step() != nil {
			return nil, fj.error("event truncated in mid-string")
		}
	}
}

func (fj *flattenJSON) readStringValWithEscapes(nameStart int) ([]byte, error) {
	// pointing at '"'
	val := []byte{'"'}
	var err error
	from := nameStart + 1
	for {
		ch := fj.event[from]
		if ch == '"' {
			fj.eventIndex = from
			val = append(val, '"')
			return val, nil
		} else if ch == '\\' {
			var unescaped []byte
			unescaped, from, err = fj.readTextWithEscapes(from)
			if err != nil {
				return nil, err
			}
			val = append(val, unescaped...)
		} else if ch <= 0x1f || ch >= byte(byteCeiling) {
			return nil, fj.error(fmt.Sprintf("illegal UTF-8 byte %x in string value", ch))
		} else {
			val = append(val, ch)
		}
		from++
		if from == len(fj.event) {
			return nil, fj.error("premature end of event")
		}
	}
}

// we're positioned at the " that marks the start of an object member name
//  ideally, we'd like to construct the member name as just a slice of the event buffer,
//  but will have to find a new home for it if it has JSON \-escapes
func (fj *flattenJSON) readMemberName() ([]byte, error) {
	// member name starts after "
	if fj.step() != nil {
		return nil, fj.error("premature end of event")
	}
	nameStart := fj.eventIndex
	for {
		ch := fj.ch()
		if ch == '"' {
			return fj.event[nameStart:fj.eventIndex], nil
		} else if ch == '\\' {
			name, err := fj.readMemberNameWithEscapes(nameStart)
			return name, err
		} else if ch <= 0x1f || ch >= byte(byteCeiling) {
			return nil, fj.error(fmt.Sprintf("illegal UTF-8 byte %x in field name", ch))
		}
		if fj.step() != nil {
			return nil, fj.error("premature end of event")
		}
	}
}

func (fj *flattenJSON) readMemberNameWithEscapes(nameStart int) ([]byte, error) {
	var err error
	var memberName []byte
	from := nameStart
	for {
		ch := fj.event[from]
		if ch == '"' {
			fj.eventIndex = from
			return memberName, nil
		} else if ch <= 0x1f || ch >= byte(byteCeiling) {
			return nil, fj.error(fmt.Sprintf("illegal UTF-8 byte %x in field name", ch))
		} else if ch == '\\' {
			var unescaped []byte
			unescaped, from, err = fj.readTextWithEscapes(from)
			if err != nil {
				return nil, err
			}
			memberName = append(memberName, unescaped...)
		} else {
			memberName = append(memberName, ch)
		}
		from++
		if from == len(fj.event) {
			return nil, fj.error("premature end of event")
		}
	}
}

// readTextWithEscapes is invoked when the next-level-up reader sees "\". JSON escape handling is simple and
//  mechanical except for \u utf-16 escapes, which get their own func.
func (fj *flattenJSON) readTextWithEscapes(from int) ([]byte, int, error) {
	// pointing at \
	unescaped := make([]byte, 1)
	var err error
	from++
	if from == len(fj.event) {
		return nil, 0, fj.error("premature end of event")
	}
	switch fj.event[from] {
	case '"':
		unescaped[0] = '"'
	case '\\':
		unescaped[0] = '\\'
	case '/':
		unescaped[0] = '/'
	case 'b':
		unescaped[0] = 8
	case 'f':
		unescaped[0] = 0xc
	case 'n':
		unescaped[0] = '\n'
	case 'r':
		unescaped[0] = '\r'
	case 't':
		unescaped[0] = '\t'
	case 'u':
		unescaped, from, err = fj.readHexUTF16(from)
		if err != nil {
			return nil, 0, err
		}
	default:
		return nil, 0, fj.error("malformed \\-escape in text")
	}

	return unescaped, from, nil
}

// readHexUTF16 is invoked when the next-level-up reader sees \u. What JSON escapes encode are 16-bit UTF-16
//  codepoints. For this reason, to turn them into UTF-8 []byte slices, you need to process any adjacent escapes
//  as a package (not gonna explain why here, go look up "UTF-16 surrogates" if you want to know. So we build up
//  a []uint16 slice and then use the builtin utf16 libraries to turn that into a []rune which we have to
//  turn into a string to extract []byte.
// the from is the offset in fj.event. We return the UTF-8 byte slice, the new setting for fj.eventIndex after
//  reading the escapes, and an error if the escape syntax is busted.
func (fj *flattenJSON) readHexUTF16(from int) ([]byte, int, error) {
	// in the case that there are multiple \uXXXX in a row, we need to read all of them because some of them
	//  might be surrogate pairs. So, back up to point at the first \
	var codepoints []uint16
	var runes []rune
	from-- // point at the \ before the u
	var hexDigitCount int
	state := startEscapeState
	for {
		ch := fj.event[from]
		switch state {
		case startEscapeState:
			switch ch {
			case '\\':
				state = wantEscapeUState
			default:
				runes = utf16.Decode(codepoints)
				return []byte(string(runes)), from - 1, nil
			}
		case wantEscapeUState:
			switch ch {
			case 'u':
				state = readHexDigitState
				hexDigitCount = 0
			default:
				runes = utf16.Decode(codepoints)
				return []byte(string(runes)), from - 1, nil
			}
		case readHexDigitState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'A', 'b', 'B', 'c', 'C', 'd', 'D', 'e', 'E', 'f', 'F':
				hexDigitCount++
				if hexDigitCount == 4 {
					hexString := string(fj.event[from-3 : from+1])
					r, _ := strconv.ParseUint(hexString, 16, 16)
					codepoints = append(codepoints, uint16(r))
					state = startEscapeState
				}
			default:
				fj.eventIndex = from
				return nil, 0, fj.error("four hex digits required after \\u")
			}
		}
		from++
		if from == len(fj.event) {
			fj.eventIndex = from
			return nil, 0, fj.error("event truncated in \\u escape")
		}
	}
}

// pathForChild does what the name says.  Since this is likely to be written into the flattened fields,
//  in many circumstances it needs its own copy of the path info
func pathForChild(pathSoFar []byte, nextSegment []byte) []byte {
	var mp []byte
	if len(pathSoFar) == 0 {
		mp = make([]byte, len(nextSegment))
		copy(mp, nextSegment)
	} else {
		mp = make([]byte, 0, len(pathSoFar)+1+len(nextSegment))
		mp = append(mp, pathSoFar...)
		mp = append(mp, '\n')
		mp = append(mp, nextSegment...)
	}
	return mp
}

// storeArrayElementField adds a field to be returned to the Flatten caller, straightforward except for the field needs its
//  own snapshot of the array-trail data, because it'll be different for each array element
//  NOTE: The profiler says this is the most expensive function in the whole matchesForJSONEvent universe, presumably
//   because of the necessity to construct a new arrayTrail for each element.
func (fj *flattenJSON) storeArrayElementField(path []byte, val []byte) {
	f := Field{Path: path, ArrayTrail: make([]ArrayPos, len(fj.arrayTrail)), Val: val}
	copy(f.ArrayTrail, fj.arrayTrail)
	fj.fields = append(fj.fields, f)
}

func (fj *flattenJSON) storeObjectMemberField(path []byte, arrayTrail []ArrayPos, val []byte) {
	fj.fields = append(fj.fields, Field{Path: path, ArrayTrail: arrayTrail, Val: val})
}

func (fj *flattenJSON) enterArray() {
	fj.arrayCount++
	fj.arrayTrail = append(fj.arrayTrail, ArrayPos{fj.arrayCount, 0})
}

func (fj *flattenJSON) leaveArray() {
	fj.arrayTrail = fj.arrayTrail[:len(fj.arrayTrail)-1]
}

func (fj *flattenJSON) stepOneArrayElement() {
	fj.arrayTrail[len(fj.arrayTrail)-1].Pos++
}

// ch fetches the next byte from the event. It doesn't check array bounds,
//  so it's the caller's responsibility to ensure we haven't run off the end of the event.
func (fj *flattenJSON) ch() byte {
	return fj.event[fj.eventIndex]
}

// step advances the event pointer and returns an error if you've run off the end of the event
func (fj *flattenJSON) step() error {
	fj.eventIndex++
	if fj.eventIndex < len(fj.event) {
		return nil
	}
	return fj.error("premature end of event")
}

func (fj *flattenJSON) error(message string) error {
	// let's be helpful and let them know where the error is
	lineNum := 1
	lastLineStart := 0
	for i := 0; i < fj.eventIndex; i++ {
		if fj.event[i] == '\n' {
			lineNum++
			lastLineStart = i
		}
	}
	return fmt.Errorf("at line %d col %d: %s", lineNum, fj.eventIndex-lastLineStart, message)
}
