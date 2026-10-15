package quamina

import (
	"cmp"
	"encoding/binary"
	"slices"
	"time"
)

// This file implements the batch compiler behind AddPatterns.
//
// AddPattern grows the automaton one Pattern at a time. To keep that possible, it gives every value of every
// field its own next fieldMatcher and then adds the rest of the Pattern to each of them, so a Pattern with
// three fields of 600, 44, and 6 values produces 600*44*6 fieldMatchers. It also merges each new value into
// the existing value automaton, unpacking and repacking smallTables along the way.
//
// The batch compiler instead works on an intermediate, immutable representation:
//
//   - Each Pattern becomes a chain of cNodes, built from its last field to its first, so every value of a field
//     points at the same continuation. The chain is linear in the number of values in the Pattern.
//   - cNodes and cValues are hash-consed: structurally identical ones are the same object. Identity is decided
//     by an exact canonical encoding, never by a hash alone, so distinct structures can't be merged by accident.
//   - All the chains are unioned together. A union of nodes is memoized on the set of nodes being unioned, so
//     e.g. every geohash cell covered by the same set of subscriptions is computed once.
//   - The result is materialized into fieldMatchers, valueMatchers, and faStates. String, number, and prefix
//     values are built directly into a byte trie and each smallTable is packed exactly once; faStates are
//     hash-consed too, so common suffixes are shared. Other value types are built with the same FA builders
//     AddPattern uses and merged in.
//
// Because the materialized automaton shares structure, it must never be mutated in place. Once a coreMatcher
// has been built this way, all subsequent additions go through the compiler, which unions the new Patterns
// with the existing intermediate representation, materializes only the cNodes and cValues that changed, and
// atomically swaps in the new start state.

type parsedPattern struct {
	x      X
	fields []*patternField
}

// cNode is the intermediate equivalent of a fieldMatcher.
type cNode struct {
	id          int
	matchIDs    []int // sorted indexes into compactBuilder.xs
	trans       []cTrans
	existsTrue  []cLink
	existsFalse []cLink
	fm          *fieldMatcher
}

type cTrans struct {
	path  string
	value *cValue
}

type cLink struct {
	path string
	node *cNode
}

// cValue is the intermediate equivalent of a valueMatcher: a set of values, each leading to a cNode.
type cValue struct {
	id      int
	entries []cEntry // sorted by key
	vm      *valueMatcher
}

type cEntry struct {
	key  string
	val  typedVal
	node *cNode
}

type compactBuilder struct {
	root *cNode

	xIDs map[X]int
	xs   []X

	nodes  map[string]*cNode
	values map[string]*cValue
	nextID int

	states   map[string]*faState
	stateIDs map[*faState]int

	regexpSerial int
	closureBufs  *closureBuffers
	keyBuf       []byte

	// per-build memoization of unions, keyed by the sorted IDs of the inputs
	nodeUnions  map[string]*cNode
	valueUnions map[string]*cValue

	// interner sizes after the last time unreachable entries were dropped
	liveNodes  int
	liveValues int
}

func newCompactBuilder() *compactBuilder {
	return &compactBuilder{
		xIDs:        make(map[X]int),
		nodes:       make(map[string]*cNode),
		values:      make(map[string]*cValue),
		states:      make(map[string]*faState),
		stateIDs:    make(map[*faState]int),
		closureBufs: newClosureBuffers(),
	}
}

// build adds the patterns to the matcher. The caller must hold m.lock.
func (b *compactBuilder) build(m *coreMatcher, patterns []parsedPattern, buildMode MatcherBuildMode) {
	if len(patterns) == 0 {
		return
	}
	b.nodeUnions = make(map[string]*cNode)
	b.valueUnions = make(map[string]*cValue)
	b.closureBufs.reset()

	segmentsTree := m.fields().segmentsTree.copy()
	roots := make([]*cNode, 0, len(patterns)+1)
	if b.root != nil {
		roots = append(roots, b.root)
	}
	for _, p := range patterns {
		for _, field := range p.fields {
			segmentsTree.add(field.path)
		}
		roots = append(roots, b.chain(p))
	}
	root := b.unionNodes(roots)
	start := b.fieldMatcherFor(root, buildMode)

	b.root = root
	b.nodeUnions = nil
	b.valueUnions = nil
	m.updateable.Store(&coreFields{state: start, segmentsTree: segmentsTree})

	if len(b.nodes) > 2*b.liveNodes+1024 || len(b.values) > 2*b.liveValues+1024 {
		b.dropUnreachable()
	}
}

func (b *compactBuilder) xID(x X) int {
	id, ok := b.xIDs[x]
	if !ok {
		id = len(b.xs)
		b.xIDs[x] = id
		b.xs = append(b.xs, x)
	}
	return id
}

// chain builds the cNodes for a single Pattern, whose fields must be sorted by path.
func (b *compactBuilder) chain(p parsedPattern) *cNode {
	next := b.internNode(&cNode{matchIDs: []int{b.xID(p.x)}})
	for i := len(p.fields) - 1; i >= 0; i-- {
		field := p.fields[i]
		if len(field.vals) == 0 {
			continue
		}
		switch field.vals[0].vType {
		case existsTrueType:
			next = b.internNode(&cNode{existsTrue: []cLink{{path: field.path, node: next}}})
		case existsFalseType:
			next = b.internNode(&cNode{existsFalse: []cLink{{path: field.path, node: next}}})
		default:
			entries := make([]cEntry, 0, len(field.vals))
			for _, val := range field.vals {
				entries = append(entries, cEntry{key: b.valKey(val), val: val, node: next})
			}
			slices.SortFunc(entries, func(a, b cEntry) int { return cmp.Compare(a.key, b.key) })
			entries = slices.CompactFunc(entries, func(a, b cEntry) bool { return a.key == b.key })
			value := b.internValue(&cValue{entries: entries})
			next = b.internNode(&cNode{trans: []cTrans{{path: field.path, value: value}}})
		}
	}
	return next
}

// valKey returns a string that is equal for two typedVals exactly when they match the same field values.
// Parsed regexps don't retain their source text, so each one gets a unique key; that's correct, but means
// identical regexps in different Patterns aren't merged.
func (b *compactBuilder) valKey(val typedVal) string {
	key := []byte{byte(val.vType)}
	if val.vType == regexpType {
		b.regexpSerial++
		return string(binary.AppendUvarint(key, uint64(b.regexpSerial)))
	}
	key = binary.AppendUvarint(key, uint64(len(val.val)))
	key = append(key, val.val...)
	for _, item := range val.list {
		key = binary.AppendUvarint(key, uint64(len(item)))
		key = append(key, item...)
	}
	return string(key)
}

func (b *compactBuilder) nodeKey(n *cNode) []byte {
	k := b.keyBuf[:0]
	k = binary.AppendUvarint(k, uint64(len(n.matchIDs)))
	for _, id := range n.matchIDs {
		k = binary.AppendUvarint(k, uint64(id))
	}
	k = binary.AppendUvarint(k, uint64(len(n.trans)))
	for _, t := range n.trans {
		k = binary.AppendUvarint(k, uint64(len(t.path)))
		k = append(k, t.path...)
		k = binary.AppendUvarint(k, uint64(t.value.id))
	}
	for _, links := range [][]cLink{n.existsTrue, n.existsFalse} {
		k = binary.AppendUvarint(k, uint64(len(links)))
		for _, l := range links {
			k = binary.AppendUvarint(k, uint64(len(l.path)))
			k = append(k, l.path...)
			k = binary.AppendUvarint(k, uint64(l.node.id))
		}
	}
	b.keyBuf = k
	return k
}

func (b *compactBuilder) valueKey(v *cValue) []byte {
	k := b.keyBuf[:0]
	k = binary.AppendUvarint(k, uint64(len(v.entries)))
	for _, e := range v.entries {
		k = binary.AppendUvarint(k, uint64(len(e.key)))
		k = append(k, e.key...)
		k = binary.AppendUvarint(k, uint64(e.node.id))
	}
	b.keyBuf = k
	return k
}

func (b *compactBuilder) internNode(n *cNode) *cNode {
	k := b.nodeKey(n)
	if existing, ok := b.nodes[string(k)]; ok {
		return existing
	}
	b.nextID++
	n.id = b.nextID
	b.nodes[string(k)] = n
	return n
}

func (b *compactBuilder) internValue(v *cValue) *cValue {
	k := b.valueKey(v)
	if existing, ok := b.values[string(k)]; ok {
		return existing
	}
	b.nextID++
	v.id = b.nextID
	b.values[string(k)] = v
	return v
}

func idsKey(ids []int) string {
	k := make([]byte, 0, len(ids)*3)
	for _, id := range ids {
		k = binary.AppendUvarint(k, uint64(id))
	}
	return string(k)
}

// unionNodes returns the cNode that matches exactly what any of the argument cNodes match.
func (b *compactBuilder) unionNodes(nodes []*cNode) *cNode {
	slices.SortFunc(nodes, func(x, y *cNode) int { return cmp.Compare(x.id, y.id) })
	nodes = slices.Compact(nodes)
	if len(nodes) == 1 {
		return nodes[0]
	}
	ids := make([]int, len(nodes))
	for i, n := range nodes {
		ids[i] = n.id
	}
	memoKey := idsKey(ids)
	if n, ok := b.nodeUnions[memoKey]; ok {
		return n
	}

	union := &cNode{}
	transByPath := make(map[string][]*cValue)
	trueByPath := make(map[string][]*cNode)
	falseByPath := make(map[string][]*cNode)
	for _, n := range nodes {
		union.matchIDs = append(union.matchIDs, n.matchIDs...)
		for _, t := range n.trans {
			transByPath[t.path] = append(transByPath[t.path], t.value)
		}
		for _, l := range n.existsTrue {
			trueByPath[l.path] = append(trueByPath[l.path], l.node)
		}
		for _, l := range n.existsFalse {
			falseByPath[l.path] = append(falseByPath[l.path], l.node)
		}
	}
	slices.Sort(union.matchIDs)
	union.matchIDs = slices.Compact(union.matchIDs)
	for _, path := range sortedKeys(transByPath) {
		union.trans = append(union.trans, cTrans{path: path, value: b.unionValues(transByPath[path])})
	}
	for _, path := range sortedKeys(trueByPath) {
		union.existsTrue = append(union.existsTrue, cLink{path: path, node: b.unionNodes(trueByPath[path])})
	}
	for _, path := range sortedKeys(falseByPath) {
		union.existsFalse = append(union.existsFalse, cLink{path: path, node: b.unionNodes(falseByPath[path])})
	}

	union = b.internNode(union)
	b.nodeUnions[memoKey] = union
	return union
}

// unionValues returns the cValue whose entries are the union of the arguments' entries; where more than one
// argument has an entry for the same value, the entry leads to the union of their cNodes.
func (b *compactBuilder) unionValues(values []*cValue) *cValue {
	slices.SortFunc(values, func(x, y *cValue) int { return cmp.Compare(x.id, y.id) })
	values = slices.Compact(values)
	if len(values) == 1 {
		return values[0]
	}
	ids := make([]int, len(values))
	for i, v := range values {
		ids[i] = v.id
	}
	memoKey := idsKey(ids)
	if v, ok := b.valueUnions[memoKey]; ok {
		return v
	}

	type pending struct {
		entry cEntry
		nodes []*cNode
	}
	size := 0
	for _, v := range values {
		size = max(size, len(v.entries))
	}
	byKey := make(map[string]int, size)
	merged := make([]pending, 0, size)
	for _, v := range values {
		for _, e := range v.entries {
			i, ok := byKey[e.key]
			if !ok {
				i = len(merged)
				byKey[e.key] = i
				merged = append(merged, pending{entry: e})
			}
			merged[i].nodes = append(merged[i].nodes, e.node)
		}
	}
	union := &cValue{entries: make([]cEntry, len(merged))}
	for i, p := range merged {
		entry := p.entry
		if len(p.nodes) > 1 {
			entry.node = b.unionNodes(p.nodes)
		}
		union.entries[i] = entry
	}
	slices.SortFunc(union.entries, func(x, y cEntry) int { return cmp.Compare(x.key, y.key) })

	union = b.internValue(union)
	b.valueUnions[memoKey] = union
	return union
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// fieldMatcherFor materializes a cNode. Materialized structures are shared and immutable, so empty maps are
// left nil; they're only ever read.
func (b *compactBuilder) fieldMatcherFor(n *cNode, buildMode MatcherBuildMode) *fieldMatcher {
	if n.fm != nil {
		return n.fm
	}
	fields := &fmFields{}
	if len(n.matchIDs) > 0 {
		fields.matches = make([]X, len(n.matchIDs))
		for i, id := range n.matchIDs {
			fields.matches[i] = b.xs[id]
		}
	}
	if len(n.trans) > 0 {
		fields.transitions = make(map[string]*valueMatcher, len(n.trans))
		for _, t := range n.trans {
			fields.transitions[t.path] = b.valueMatcherFor(t.value, buildMode)
		}
	}
	if len(n.existsTrue) > 0 {
		fields.existsTrue = make(map[string]*fieldMatcher, len(n.existsTrue))
		for _, l := range n.existsTrue {
			fields.existsTrue[l.path] = b.fieldMatcherFor(l.node, buildMode)
		}
	}
	if len(n.existsFalse) > 0 {
		fields.existsFalse = make(map[string]*fieldMatcher, len(n.existsFalse))
		for _, l := range n.existsFalse {
			fields.existsFalse[l.path] = b.fieldMatcherFor(l.node, buildMode)
		}
	}
	fm := &fieldMatcher{}
	fm.update(fields)
	n.fm = fm
	return fm
}

func (b *compactBuilder) valueMatcherFor(v *cValue, buildMode MatcherBuildMode) *valueMatcher {
	if v.vm != nil {
		return v.vm
	}
	fields := &vmFields{}
	if len(v.entries) == 1 && (v.entries[0].val.vType == stringType || v.entries[0].val.vType == literalType) {
		fields.singletonMatch = []byte(v.entries[0].val.val)
		fields.singletonTransition = b.fieldMatcherFor(v.entries[0].node, buildMode)
	} else {
		trie := &byteTrie{}
		var others []cEntry
		tzTries := make(map[string]*byteTrie)
		tzLocs := make(map[string]*time.Location)
		for _, e := range v.entries {
			switch e.val.vType {
			case timeWindowType:
				spec := timeWindowFromVal(e.val.val)
				tzTrie := tzTries[spec.tz]
				if tzTrie == nil {
					tzTrie = &byteTrie{}
					tzTries[spec.tz] = tzTrie
					tzLocs[spec.tz] = spec.loc
				}
				for _, p := range spec.prefixes {
					tzTrie.insert(p, e.node, false)
				}
			case numericRangeType:
				fa := makeNumericRangeFA(numericRangeFromVal(e.val.val), b.fieldMatcherFor(e.node, buildMode))
				if fields.numeric == nil {
					fields.numeric = fa
				} else {
					fields.numeric = mergeStartStates(fields.numeric, fa, sharedNullPrinter)
				}
			case stringType, literalType:
				trie.insert([]byte(e.val.val), e.node, true)
			case numberType:
				fields.hasNumbers = true
				trie.insert([]byte(e.val.val), e.node, true)
				if qNum, err := qNumFromBytes([]byte(e.val.val)); err == nil {
					trie.insert(qNum, e.node, true)
				}
			case prefixType:
				// skip the closing quote
				trie.insert([]byte(e.val.val[:len(e.val.val)-1]), e.node, false)
			default:
				others = append(others, e)
			}
		}
		var start *faState
		if len(trie.children) > 0 {
			start = b.stateForTrie(trie, buildMode)
		}
		for _, e := range others {
			fa, nondeterministic := b.makeFA(e, buildMode)
			fields.isNondeterministic = fields.isNondeterministic || nondeterministic
			if start == nil {
				start = fa
			} else {
				start = mergeStartStates(start, fa, sharedNullPrinter)
			}
		}
		if fields.isNondeterministic {
			epsilonClosureInto(start, b.closureBufs)
			if buildMode == BuiltForSpeed {
				start = nfa2Dfa(start)
				fields.isNondeterministic = false
			}
		}
		fields.start = start
		for _, tz := range sortedKeys(tzTries) {
			fields.timeZones = append(fields.timeZones, tzMatcher{tz: tz, loc: tzLocs[tz], start: b.stateForTrie(tzTries[tz], buildMode)})
		}
	}
	vm := &valueMatcher{}
	vm.update(fields)
	v.vm = vm
	return vm
}

// makeFA builds the automaton for a value type that isn't compiled into the byte trie, using the same
// builders as AddPattern, and points its exit at the materialized cNode.
func (b *compactBuilder) makeFA(e cEntry, buildMode MatcherBuildMode) (*faState, bool) {
	valBytes := []byte(e.val.val)
	var fa *faState
	var next *fieldMatcher
	nondeterministic := false
	switch e.val.vType {
	case anythingButType:
		fa, next = makeMultiAnythingButFA(e.val.list)
	case shellStyleType:
		fa, next = makeShellStyleFA(valBytes, sharedNullPrinter)
		nondeterministic = true
	case wildcardType:
		fa, next = makeWildCardFA(valBytes, sharedNullPrinter)
		nondeterministic = true
	case monocaseType:
		fa, next = makeMonocaseFA(valBytes, sharedNullPrinter)
	case regexpType:
		fa, next = makeRegexpNFA(e.val.parsedRegexp, sharedNullPrinter)
		nondeterministic = fa.table.isNondeterministic()
	default:
		panic("unknown value type")
	}
	// The builders end in a fresh fieldMatcher; sharing the materialized cNode's fields makes it equivalent.
	next.update(b.fieldMatcherFor(e.node, buildMode).fields())
	return fa, nondeterministic
}

// byteTrie collects the byte sequences of the values compiled directly into a value automaton.
type byteTrie struct {
	bytes    []byte
	children []*byteTrie
	prefixes []*cNode // values that match once this point is reached
	exacts   []*cNode // values that match if the field value ends here
}

func (t *byteTrie) insert(val []byte, node *cNode, exact bool) {
	for _, c := range val {
		i := slices.Index(t.bytes, c)
		if i < 0 {
			i = len(t.bytes)
			t.bytes = append(t.bytes, c)
			t.children = append(t.children, &byteTrie{})
		}
		t = t.children[i]
	}
	if exact {
		t.exacts = append(t.exacts, node)
	} else {
		t.prefixes = append(t.prefixes, node)
	}
}

func (b *compactBuilder) stateForTrie(t *byteTrie, buildMode MatcherBuildMode) *faState {
	order := make([]int, len(t.bytes))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(x, y int) int { return cmp.Compare(t.bytes[x], t.bytes[y]) })

	indices := make([]byte, 0, len(order)+1)
	steps := make([]*faState, 0, len(order)+1)
	for _, i := range order {
		indices = append(indices, t.bytes[i])
		steps = append(steps, b.stateForTrie(t.children[i], buildMode))
	}
	if len(t.exacts) > 0 {
		indices = append(indices, valueTerminator)
		steps = append(steps, b.internState(nil, nil, t.exacts, buildMode))
	}
	return b.internState(indices, steps, t.prefixes, buildMode)
}

// internState returns a shared faState with the given byte transitions, whose fieldTransitions lead to the
// materialized cNodes.
func (b *compactBuilder) internState(indices []byte, steps []*faState, nodes []*cNode, buildMode MatcherBuildMode) *faState {
	slices.SortFunc(nodes, func(x, y *cNode) int { return cmp.Compare(x.id, y.id) })
	nodes = slices.Compact(nodes)

	k := b.keyBuf[:0]
	for i, index := range indices {
		k = append(k, index)
		k = binary.AppendUvarint(k, uint64(b.stateIDs[steps[i]]))
	}
	k = append(k, 0xff)
	for _, n := range nodes {
		k = binary.AppendUvarint(k, uint64(n.id))
	}
	b.keyBuf = k
	key := string(k) // materializing fieldTransitions below can reuse keyBuf
	if existing, ok := b.states[key]; ok {
		return existing
	}

	state := &faState{}
	if len(nodes) > 0 {
		// exactly sized, because mergeFAStates appends to fieldTransitions
		state.fieldTransitions = make([]*fieldMatcher, len(nodes))
		for i, n := range nodes {
			state.fieldTransitions[i] = b.fieldMatcherFor(n, buildMode)
		}
	}
	if len(indices) > 0 {
		state.table = makeSmallTable(nil, indices, steps)
	} else {
		state.table = newSmallTable()
	}
	b.states[key] = state
	b.stateIDs[state] = len(b.stateIDs) + 1
	return state
}

// dropUnreachable removes interned entries that are no longer reachable from the root, such as the
// per-Pattern chains consumed by unions. The faState interner is simply cleared; later builds won't share
// suffixes with existing states, which costs memory but not correctness.
func (b *compactBuilder) dropUnreachable() {
	nodes := make(map[string]*cNode)
	values := make(map[string]*cValue)
	seenNodes := make(map[*cNode]bool)
	seenValues := make(map[*cValue]bool)
	var walkNode func(n *cNode)
	walkNode = func(n *cNode) {
		if seenNodes[n] {
			return
		}
		seenNodes[n] = true
		for _, t := range n.trans {
			if !seenValues[t.value] {
				seenValues[t.value] = true
				for _, e := range t.value.entries {
					walkNode(e.node)
				}
				values[string(b.valueKey(t.value))] = t.value
			}
		}
		for _, l := range n.existsTrue {
			walkNode(l.node)
		}
		for _, l := range n.existsFalse {
			walkNode(l.node)
		}
		nodes[string(b.nodeKey(n))] = n
	}
	if b.root != nil {
		walkNode(b.root)
	}
	b.nodes = nodes
	b.values = values
	b.liveNodes = len(nodes)
	b.liveValues = len(values)
	b.states = make(map[string]*faState)
	b.stateIDs = make(map[*faState]int)
}
