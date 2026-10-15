package quamina

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// A time-window pattern matches a timestamp field whose local wall-clock time, in a given time zone, falls
// within a weekly window:
//
//	{"timestamp": [{"time-window": {"tz": "America/Los_Angeles", "days": ["mon", "fri"], "start": "08:00", "end": "17:00"}}]}
//
// "tz" is an IANA time zone name and is required. "days" lists the days on which the window starts and
// defaults to every day. "start" is inclusive and "end" exclusive, both "HH:MM"; "end" may be "24:00", and
// an end at or before the start means the window runs past midnight into the following day.
//
// Event timestamps may be numbers, which are read as Unix epoch milliseconds, or RFC 3339 strings.
//
// Local times are encoded as the five-digit minute of the week, Monday 00:00 being 00000, so that each
// window is a set of decimal prefixes and compiles into an ordinary deterministic automaton. There is one
// automaton per time zone, and matching converts the event time to each zone's wall-clock time, so
// daylight-saving changes are handled exactly.

const (
	minutesPerDay  = 24 * 60
	minutesPerWeek = 7 * minutesPerDay
	weekDigits     = 5
)

type timeWindowSpec struct {
	tz       string
	loc      *time.Location
	prefixes [][]byte
}

// tzMatcher is the automaton for the time-window values in one time zone on one field.
type tzMatcher struct {
	tz    string
	loc   *time.Location
	start *faState
}

var weekdayNames = map[string]int{
	"mon": 0, "monday": 0,
	"tue": 1, "tuesday": 1,
	"wed": 2, "wednesday": 2,
	"thu": 3, "thursday": 3,
	"fri": 4, "friday": 4,
	"sat": 5, "saturday": 5,
	"sun": 6, "sunday": 6,
}

var locations sync.Map // map[string]*time.Location

func loadLocation(name string) (*time.Location, error) {
	if loc, ok := locations.Load(name); ok {
		return loc.(*time.Location), nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	locations.Store(name, loc)
	return loc, nil
}

func readTimeWindowSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	pathVals = valsIn
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	if d, ok := t.(json.Delim); !ok || d != '{' {
		err = errors.New("value for 'time-window' must be an object")
		return
	}

	var tz, start, end string
	var days []int
	for {
		t, err = pb.jd.Token()
		if err != nil {
			return
		}
		if _, ok := t.(json.Delim); ok {
			// has to be } or the tokenizer would have thrown an error
			break
		}
		key := t.(string)
		switch key {
		case "tz", "start", "end":
			var s string
			if s, err = readString(pb, key); err != nil {
				return
			}
			switch key {
			case "tz":
				tz = s
			case "start":
				start = s
			case "end":
				end = s
			}
		case "days":
			if days, err = readDays(pb); err != nil {
				return
			}
		default:
			err = fmt.Errorf("unrecognized field %q in 'time-window'", key)
			return
		}
	}

	spec, err := newTimeWindowSpec(tz, days, start, end)
	if err != nil {
		return
	}
	canonical := spec.canonical()
	timeWindows.LoadOrStore(canonical, spec)
	pathVals = append(pathVals, typedVal{vType: timeWindowType, val: canonical})

	// has to be } or tokenizer will throw error
	_, err = pb.jd.Token()
	return
}

func readString(pb *patternBuild, key string) (string, error) {
	t, err := pb.jd.Token()
	if err != nil {
		return "", err
	}
	s, ok := t.(string)
	if !ok {
		return "", fmt.Errorf("'time-window' field %q must be a string", key)
	}
	return s, nil
}

func readDays(pb *patternBuild) ([]int, error) {
	t, err := pb.jd.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := t.(json.Delim); !ok || d != '[' {
		return nil, errors.New("'time-window' field \"days\" must be an array")
	}
	days := []int{}
	for {
		t, err = pb.jd.Token()
		if err != nil {
			return nil, err
		}
		if _, ok := t.(json.Delim); ok {
			// has to be ]
			return days, nil
		}
		name, ok := t.(string)
		day, known := weekdayNames[strings.ToLower(name)]
		if !ok || !known {
			return nil, fmt.Errorf("'time-window' day %v is not a day of the week", t)
		}
		days = append(days, day)
	}
}

func parseClock(s string, allow24 bool) (int, error) {
	h, m, ok := strings.Cut(s, ":")
	hour, errH := strconv.Atoi(h)
	minute, errM := strconv.Atoi(m)
	switch {
	case !ok || !isTwoDigits(h) || !isTwoDigits(m) || errH != nil || errM != nil:
	case allow24 && hour == 24 && minute == 0:
		return minutesPerDay, nil
	case hour >= 0 && hour < 24 && minute >= 0 && minute < 60:
		return hour*60 + minute, nil
	}
	return 0, fmt.Errorf("'time-window' time %q must be HH:MM", s)
}

func isTwoDigits(s string) bool {
	return len(s) == 2 && s[0] >= '0' && s[0] <= '9' && s[1] >= '0' && s[1] <= '9'
}

type minuteRange struct{ lo, hi int } // [lo, hi) in minutes of the week

func newTimeWindowSpec(tz string, days []int, start, end string) (*timeWindowSpec, error) {
	if tz == "" {
		return nil, errors.New("'time-window' requires \"tz\"")
	}
	loc, err := loadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("'time-window' time zone %q: %w", tz, err)
	}
	if start == "" || end == "" {
		return nil, errors.New("'time-window' requires \"start\" and \"end\"")
	}
	startMin, err := parseClock(start, false)
	if err != nil {
		return nil, err
	}
	endMin, err := parseClock(end, true)
	if err != nil {
		return nil, err
	}
	if startMin == endMin {
		return nil, errors.New("'time-window' start and end must differ; use 00:00 to 24:00 for a whole day")
	}
	if endMin < startMin {
		endMin += minutesPerDay
	}
	if days == nil {
		days = []int{0, 1, 2, 3, 4, 5, 6}
	}
	if len(days) == 0 {
		return nil, errors.New("'time-window' \"days\" must not be empty")
	}

	var ranges []minuteRange
	for _, day := range days {
		lo, hi := day*minutesPerDay+startMin, day*minutesPerDay+endMin
		if hi > minutesPerWeek {
			ranges = append(ranges, minuteRange{lo, minutesPerWeek}, minuteRange{0, hi - minutesPerWeek})
		} else {
			ranges = append(ranges, minuteRange{lo, hi})
		}
	}
	ranges = mergeRanges(ranges)

	spec := &timeWindowSpec{tz: tz, loc: loc}
	for _, r := range ranges {
		spec.prefixes = appendRangePrefixes(spec.prefixes, r.lo, r.hi)
	}
	return spec, nil
}

func mergeRanges(ranges []minuteRange) []minuteRange {
	slices.SortFunc(ranges, func(a, b minuteRange) int { return a.lo - b.lo })
	merged := ranges[:1]
	for _, r := range ranges[1:] {
		last := &merged[len(merged)-1]
		if r.lo <= last.hi {
			last.hi = max(last.hi, r.hi)
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

// appendRangePrefixes appends the decimal prefixes of the weekDigits-wide numbers in [lo, hi), using the
// largest aligned power-of-ten blocks.
func appendRangePrefixes(prefixes [][]byte, lo, hi int) [][]byte {
	for lo < hi {
		digits, block := 0, 1
		for digits < weekDigits && lo%(block*10) == 0 && lo+block*10 <= hi {
			digits++
			block *= 10
		}
		prefixes = append(prefixes, fmt.Appendf(nil, "%0*d", weekDigits, lo)[:weekDigits-digits])
		lo += block
	}
	return prefixes
}

var timeWindows sync.Map // map[string]*timeWindowSpec, keyed by canonical form

// timeWindowFromVal returns the spec for the canonical form held in a time-window typedVal's val.
func timeWindowFromVal(val string) *timeWindowSpec {
	if spec, ok := timeWindows.Load(val); ok {
		return spec.(*timeWindowSpec)
	}
	fields := strings.Fields(val)
	loc, err := loadLocation(fields[0])
	if err != nil {
		panic("time-window with unloadable time zone " + fields[0])
	}
	spec := &timeWindowSpec{tz: fields[0], loc: loc}
	for _, p := range fields[1:] {
		spec.prefixes = append(spec.prefixes, []byte(p))
	}
	actual, _ := timeWindows.LoadOrStore(val, spec)
	return actual.(*timeWindowSpec)
}

// canonical returns a string that is equal for two specs exactly when they match the same timestamps; the
// spec can be rebuilt from it.
func (s *timeWindowSpec) canonical() string {
	var sb strings.Builder
	sb.WriteString(s.tz)
	for _, p := range s.prefixes {
		sb.WriteByte(' ')
		sb.Write(p)
	}
	return sb.String()
}

// makePrefixSetFA builds a deterministic automaton that transitions to next on reaching any of the
// prefixes, which must be distinct.
func makePrefixSetFA(prefixes [][]byte, next *fieldMatcher) *faState {
	state := &faState{table: newSmallTable()}
	byFirst := make(map[byte][][]byte)
	for _, p := range prefixes {
		if len(p) == 0 {
			state.fieldTransitions = []*fieldMatcher{next}
			continue
		}
		byFirst[p[0]] = append(byFirst[p[0]], p[1:])
	}
	if len(byFirst) == 0 {
		return state
	}
	bytes := make([]byte, 0, len(byFirst))
	for b := range byFirst {
		bytes = append(bytes, b)
	}
	slices.Sort(bytes)
	steps := make([]*faState, len(bytes))
	for i, b := range bytes {
		steps[i] = makePrefixSetFA(byFirst[b], next)
	}
	state.table = makeSmallTable(nil, bytes, steps)
	return state
}

// addTimeWindow returns a copy of tzs with the time window, leading to next, added.
func addTimeWindow(tzs []tzMatcher, spec *timeWindowSpec, next *fieldMatcher) []tzMatcher {
	fa := makePrefixSetFA(spec.prefixes, next)
	tzs = slices.Clone(tzs)
	for i := range tzs {
		if tzs[i].tz == spec.tz {
			tzs[i].start = mergeStartStates(tzs[i].start, fa, sharedNullPrinter)
			return tzs
		}
	}
	return append(tzs, tzMatcher{tz: spec.tz, loc: spec.loc, start: fa})
}

// eventTime reads a timestamp from an event field: a number of Unix epoch milliseconds or an RFC 3339 string.
func eventTime(field *Field) (time.Time, bool) {
	val := field.Val
	if field.IsNumber {
		ms, ok := parseEpochMillis(val)
		return time.UnixMilli(ms), ok
	}
	if len(val) < 2 || val[0] != '"' || val[len(val)-1] != '"' {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, string(val[1:len(val)-1]))
	return t, err == nil
}

func parseEpochMillis(val []byte) (int64, bool) {
	if len(val) == 0 || len(val) > 18 {
		return 0, false
	}
	neg := val[0] == '-'
	if neg {
		val = val[1:]
		if len(val) == 0 {
			return 0, false
		}
	}
	var ms int64
	for _, c := range val {
		if c < '0' || c > '9' {
			return 0, false
		}
		ms = ms*10 + int64(c-'0')
	}
	if neg {
		ms = -ms
	}
	return ms, true
}

// timeTransitions appends the transitions of the time-window automata matching the event field's timestamp.
func timeTransitions(tzs []tzMatcher, field *Field, transitions []*fieldMatcher, bufs *nfaBuffers) []*fieldMatcher {
	t, ok := eventTime(field)
	if !ok {
		return transitions
	}
	for i := range tzs {
		local := t.In(tzs[i].loc)
		hour, minute, _ := local.Clock()
		minuteOfWeek := ((int(local.Weekday())+6)%7)*minutesPerDay + hour*60 + minute
		buf := bufs.timeBuf[:]
		for d := weekDigits - 1; d >= 0; d-- {
			buf[d] = byte('0' + minuteOfWeek%10)
			minuteOfWeek /= 10
		}
		transitions = traverseDFA(tzs[i].start, buf, transitions)
	}
	return transitions
}
