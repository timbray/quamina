package quamina

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf16"
)

// flattenJSON is a custom non-general-purpose JSON parser whose object is to implement Flattener and produce a []Field
// list from a JSON object.  This could be done (and originally was) with the built-in encoding/json, but the
// performance was unsatisfactory (99% of time spent parsing events < 1% matching them). The profiler suggests
// that the performance issue was mostly due to excessive memory allocation.
// If we assume that the event is immutable while we're working, then all the pieces of it that constitute
// the fields & values can be represented as []byte slices using a couple of offsets into the underlying event.
// There is an exception, namely strings that contain \-prefixed JSON escapes; since we want to work with the
// actual UTF-8 bytes, this requires re-writing such strings into memory we have to allocate.
type flattenJSON struct {
	event      []byte     // event being processed, treated as immutable
	eventIndex int        // current byte index into the event
	fields     []Field    // the under-construction return value of the Flatten method
	skipping   int        // track whether we're within the scope of a segment that isn't used
	arrayTrail []ArrayPos // current array-position cookie crumbs
	arrayCount int32      // how many arrays we've seen, used in building arrayTrail
	cleanSheet bool       // initially true, don't have to call Reset()
	isSpace    [256]bool
}

// Reset a flattenJSON struct so  that it can be re-used and won't need to be reconstructed for each event
// to be flattened
func (fj *flattenJSON) reset() {
	fj.eventIndex = 0
	fj.fields = fj.fields[:0]
	fj.skipping = 0
	fj.arrayTrail = fj.arrayTrail[:0]
	fj.arrayCount = 0
}

// JSON literals
var (
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
	nullBytes  = []byte("null")
)

// errEarlyStop is used to signal the case when we've detected that we've read all the fields that appear in any pattern
// and so that we don't need to read any more
var errEarlyStop = errors.New("earlyStop")

// fjState - this is a finite state machine parser, or rather a collection of smaller FSM parsers. Some of these
// states are used in only one function, others in multiple places
type fjState int

const (
	fjStartState fjState = iota
	fjInObjectState
	fjSeekingColonState
	fjMemberValueState
	fjInArrayState
	fjAfterValueState
	fjNumberStartState
	fjNumberIntegralPartState
	fjNumberFracState
	fjNumberAfterEState
	fjNumberExpState
	fjTrailerState
	fjStartEscapeState
	fjWantEscapeUState
	fjReadHexDigitState
)

func newJSONFlattener() Flattener {
	f := &flattenJSON{fields: make([]Field, 0, 32), cleanSheet: true}
	for _, space := range []byte{' ', '\r', '\n', '\t'} {
		f.isSpace[space] = true
	}
	return f
}

func (fj *flattenJSON) Copy() Flattener {
	return newJSONFlattener()
}

// Flatten implements the Flattener interface. It assumes that the event is immutable - if you modify the event
// bytes while the matcher is running, grave disorder will ensue.
func (fj *flattenJSON) Flatten(event []byte, tracker SegmentsTreeTracker) ([]Field, error) {
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
	state := fjStartState
	for {
		ch := fj.ch()
		switch state {
		case fjStartState:
			switch {
			// single top-level object
			case ch == '{':
				err = fj.readObject(tracker)
				if err != nil {
					if errors.Is(err, errEarlyStop) {
						return fj.fields, nil
					}
					return nil, err
				}
				state = fjTrailerState

			case fj.isSpace[ch]:
			// no-op

			default:
				return nil, fj.error("not a JSON object")
			}

		// eat trailing white space, if any
		case fjTrailerState:
			if !fj.isSpace[ch] {
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

// readObject - process through a JSON object, recursing if necessary into sub-objects. pathNode is used to
// determine whether any particular object member is used, and skipping tracks that status up and down the stack.
// This is all done to allow the parser to skip child nodes which do not appear in any Patterns and thus
// minimize the cost of the Flatten call.
func (fj *flattenJSON) readObject(pathNode SegmentsTreeTracker) error {
	var err error
	state := fjInObjectState

	// eventIndex points at {
	err = fj.step()
	if err != nil {
		return err
	}

	// how many leaf states (fieldsCount) and chidStructures (nodesCount) have been mentioned in patterns?
	fieldsCount := pathNode.FieldsCount()
	nodesCount := pathNode.NodesCount()

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
		// if we've read all the nodes and fields tha have been mentioned in Patterns, we can stop reading this object
		if nodesCount == 0 && fieldsCount == 0 {
			if pathNode.IsRoot() {
				return errEarlyStop
			} else {
				return fj.leaveObject()
			}
		}

		ch := fj.ch()

		switch state {
		case fjInObjectState:
			switch {
			case fj.isSpace[ch]:
				// no-op
			case ch == '"':
				memberName, err = fj.readMemberName()
				if err != nil {
					return err
				}

				// we know the name of the next object member, use the pathNode to check if it's used
				memberIsUsed = (fj.skipping == 0) && pathNode.IsSegmentUsed(memberName)
				state = fjSeekingColonState
			case ch == '}':
				return nil
			default:
				return fj.error(fmt.Sprintf("illegal character %c in JSON object", ch))
			}
		case fjSeekingColonState:
			switch {
			case fj.isSpace[ch]:
				// no-op
			case ch == ':':
				state = fjMemberValueState
			default:
				return fj.error(fmt.Sprintf("illegal character %c while looking for colon", ch))
			}
		case fjMemberValueState:
			// bypass space between colon and value. A bit klunky but allows for immense simplification
			// TODO: Investigate if there's a more efficient way to say this, or should just trust Go compiler
			for fj.isSpace[ch] {
				if fj.step() != nil {
					return fj.error("event truncated after colon")
				}
				ch = fj.ch()
			}

			var val []byte
			isNumber := false
			switch ch {
			case '"':
				if fj.skipping > 0 || !memberIsUsed {
					err = fj.skipStringValue()
				} else {
					val, err = fj.readStringValue()
				}
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
				val, err = fj.readNumber()
				if err == nil {
					isNumber = true
				}
				isLeaf = true
			case '[':
				if !pathNode.IsSegmentUsed(memberName) {
					fj.skipping++
				}

				if fj.skipping > 0 || !memberIsUsed {
					err = fj.skipBlock('[', ']')
				} else {
					arrayPathNode, ok := pathNode.Get(memberName)
					if !ok {
						// Arrays are interesting, they can be field or node.
						// Given this case:
						//  { "geo": { "coords": [{"coordinates": [1,2,3]}] } }
						// "coords" is a node.
						// "coordinates" is a field.
						arrayPathNode = pathNode
					}

					err = fj.readArray(pathNode.PathForSegment(memberName), arrayPathNode)
				}
				if err != nil {
					return err
				}
				if !pathNode.IsSegmentUsed(memberName) {
					fj.skipping--
				}
			case '{':
				if !pathNode.IsSegmentUsed(memberName) {
					fj.skipping++
				}
				if fj.skipping > 0 || !memberIsUsed {
					err = fj.skipBlock('{', '}')
				} else {
					objectPathNode, ok := pathNode.Get(memberName)
					if !ok {
						// This can happen if we got a pattern which is doing matching on object (for example: exists on object)
						// Currently, we don't support this case, so we will skip the block.
						err = fj.skipBlock('{', '}')
					} else {
						// Traversing into node, reduce the count.
						nodesCount--

						err = fj.readObject(objectPathNode)
					}
				}
				if err != nil {
					return err
				}
				if !pathNode.IsSegmentUsed(memberName) {
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
					fj.storeObjectMemberField(pathNode.PathForSegment(memberName), arrayTrail, val, isNumber)
					fieldsCount--
				}
			}
			state = fjAfterValueState
		case fjAfterValueState:
			switch {
			case fj.isSpace[ch]:
				// no-op
			case ch == ',':
				state = fjInObjectState
			case ch == '}':
				return nil
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

// read an array in an incoming event, recursing as necessary into members. pathNode and fj.skipping are
// used to bypass elements where possible.
func (fj *flattenJSON) readArray(pathName []byte, pathNode SegmentsTreeTracker) error {
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

	state := fjInArrayState
	isLeaf := false
	for {
		ch := fj.ch()
		var val []byte // resets on each loop
		isNumber := false
		switch state {
		case fjInArrayState:
			// bypass space before element value. A bit klunky but allows for immense simplification
			for fj.isSpace[ch] {
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
				val, err = fj.readNumber()
				if err == nil {
					isNumber = true
				}
				isLeaf = true
			case '{':
				if fj.skipping == 0 {
					fj.stepOneArrayElement()
				}

				err = fj.readObject(pathNode)

				if err != nil {
					return err
				}
			case '[':
				if fj.skipping == 0 {
					fj.stepOneArrayElement()
				}
				err = fj.readArray(pathName, pathNode)
				if err != nil {
					return err
				}
			case ']':
				return nil
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
					fj.storeArrayElementField(pathName, val, isNumber)
				}
			}
			state = fjAfterValueState
		case fjAfterValueState:
			switch {
			case fj.isSpace[ch]:
				// no-op
			case ch == ']':
				return nil
			case ch == ',':
				state = fjInArrayState
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

func (fj *flattenJSON) readNumber() ([]byte, error) {
	// points at the first character in the number
	numStart := fj.eventIndex
	state := fjNumberStartState
	for {
		ch := fj.ch()
		switch state {
		case fjNumberStartState:
			switch ch {
			case '-':
				state = fjNumberIntegralPartState
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				state = fjNumberIntegralPartState
			}
		case fjNumberIntegralPartState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			case '.':
				state = fjNumberFracState
			case 'e', 'E':
				state = fjNumberAfterEState
			case ',', ']', '}', ' ', '\t', '\n', '\r':
				fj.eventIndex--
				return fj.event[numStart : fj.eventIndex+1], nil
			default:
				return nil, fj.error(fmt.Sprintf("illegal char '%c' in number", ch))
			}
		case fjNumberFracState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			case ',', ']', '}', ' ', '\t', '\n', '\r':
				fj.eventIndex--
				bytes := fj.event[numStart : fj.eventIndex+1]
				return bytes, nil
			case 'e', 'E':
				state = fjNumberAfterEState
			default:
				return nil, fj.error(fmt.Sprintf("illegal char '%c' in number", ch))
			}
		case fjNumberAfterEState:
			switch ch {
			case '-', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			default:
				return nil, fj.error(fmt.Sprintf("illegal char '%c' after 'e' in number", ch))
			}
			state = fjNumberExpState

		case fjNumberExpState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				// no-op
			case ',', ']', '}', ' ', '\t', '\n', '\r':
				fj.eventIndex--
				return fj.event[numStart : fj.eventIndex+1], nil
			default:
				return nil, fj.error(fmt.Sprintf("illegal char '%c' in exponent", ch))
			}
		}
		if fj.step() != nil {
			return nil, fj.error("event truncated in number")
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

func (fj *flattenJSON) leaveObject() error {
	for fj.eventIndex < len(fj.event) {
		ch := fj.event[fj.eventIndex]

		switch ch {
		case '"':
			err := fj.skipStringValue()
			if err != nil {
				return err
			}
		case '{', '[':
			// ch+2 is the matching closing brace, since both '}' and ']' are 2 characters away
			// from '{' and ']', respectively
			err := fj.skipBlock(ch, ch+2)
			if err != nil {
				return err
			}
		case '}':
			return nil
		}

		fj.eventIndex++
	}

	return fj.error("truncated block")
}

// used to bypass object members and array elements which are not significant to any Pattern more quickly
// than running the whole state machine.
func (fj *flattenJSON) skipBlock(openSymbol byte, closeSymbol byte) error {
	level := 0

	for fj.eventIndex < len(fj.event) {
		ch := fj.event[fj.eventIndex]

		switch ch {
		case '"':
			err := fj.skipStringValue()
			if err != nil {
				return err
			}
		case openSymbol:
			level++
		case closeSymbol:
			level--

			if level == 0 {
				return nil
			}
		}

		fj.eventIndex++
	}

	return fj.error("truncated block")
}

func (fj *flattenJSON) skipStringValue() error {
	if fj.step() != nil {
		return fj.error("event truncated in mid-string")
	}

	i := 0
	data := fj.event[fj.eventIndex:]

	for i < len(data) {
		c := data[i]

		// Ignore double slashes (double escaped values)
		if c == '\\' && i+1 < len(data) && data[i+1] == '\\' {
			i = i + 2
			continue
		}

		// Since we want to iterate until we found quote (") we need to take care
		// about escaped quotes (\"), any other escaped characters is not relevant.
		if c == '\\' && i+1 < len(data) && data[i+1] == '"' {
			i = i + 2
			continue
		}

		// If we found a quote, and it's not escaped (we check it above)
		// we can finish processing.
		if c == '"' {
			fj.eventIndex = fj.eventIndex + i
			return nil
		}

		i++
	}

	return fj.error("truncated string")
}

// we're positioned at the " that marks the start of a string value in an array or object.
// ideally, we'd like to construct the member name as just a slice of the event buffer,
// but will have to find a new home for it if it has JSON \-escapes
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
// ideally, we'd like to construct the member name as just a slice of the event buffer,
// but will have to find a new home for it if it has JSON \-escapes
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
// mechanical except for \u utf-16 escapes, which get their own func.
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
// codepoints. For this reason, to turn them into UTF-8 []byte slices, you need to process any adjacent escapes
// as a package (not gonna explain why here, go look up "UTF-16 surrogates" if you want to know. So we build up
// a []uint16 slice and then use the builtin utf16 libraries to turn that into a []rune which we have to
// turn into a string to extract []byte.
// the from is the offset in fj.event. We return the UTF-8 byte slice, the new setting for fj.eventIndex after
// reading the escapes, and an error if the escape syntax is busted.
func (fj *flattenJSON) readHexUTF16(from int) ([]byte, int, error) {
	// in the case that there are multiple \uXXXX in a row, we need to read all of them because some of them
	//  might be surrogate pairs. So, back up to point at the first \
	var codepoints []uint16
	var runes []rune
	from-- // point at the \ before the u
	var hexDigitCount int
	state := fjStartEscapeState
	for {
		ch := fj.event[from]
		switch state {
		case fjStartEscapeState:
			switch ch {
			case '\\':
				state = fjWantEscapeUState
			default:
				runes = utf16.Decode(codepoints)
				return []byte(string(runes)), from - 1, nil
			}
		case fjWantEscapeUState:
			switch ch {
			case 'u':
				state = fjReadHexDigitState
				hexDigitCount = 0
			default:
				runes = utf16.Decode(codepoints)
				return []byte(string(runes)), from - 1, nil
			}
		case fjReadHexDigitState:
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'A', 'b', 'B', 'c', 'C', 'd', 'D', 'e', 'E', 'f', 'F':
				hexDigitCount++
				if hexDigitCount == 4 {
					hexString := string(fj.event[from-3 : from+1])
					r, _ := strconv.ParseUint(hexString, 16, 16)
					codepoints = append(codepoints, uint16(r))
					state = fjStartEscapeState
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

// storeArrayElementField adds a field to be returned to the Flatten caller, straightforward except for the field needs
// its own snapshot of the array-trail data, because it'll be different for each array element
// NOTE: The profiler says this is the most expensive function in the whole matchesForJSONEvent universe, presumably
// because of the necessity to construct a new arrayTrail for each element.
func (fj *flattenJSON) storeArrayElementField(path []byte, val []byte, isNumber bool) {
	f := Field{Path: path, ArrayTrail: make([]ArrayPos, len(fj.arrayTrail)), Val: val, IsNumber: isNumber}
	copy(f.ArrayTrail, fj.arrayTrail)
	fj.fields = append(fj.fields, f)
}

func (fj *flattenJSON) storeObjectMemberField(path []byte, arrayTrail []ArrayPos, val []byte, isNumber bool) {
	fj.fields = append(fj.fields, Field{Path: path, ArrayTrail: arrayTrail, Val: val, IsNumber: isNumber})
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
// so it's the caller's responsibility to ensure we haven't run off the end of the event.
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
