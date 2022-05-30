package constants

// ByteCeiling - the automaton runs on UTF-8 bytes, which map nicely to Go's byte, which is uint8. The values
//  0xF5-0xFF can't appear in UTF-8 strings. We use 0xF5 as a value terminator, so characters F6 and higher
//  can't appear.
const ByteCeiling int = 0xf6

// ValueTerminator - whenever we're trying to match a value with a pattern that extends to the end of that
//  value, we virtually add one of these as the last character, both to the automaton and the value at run-time.
//  This simplifies things because you don't have to treat absolute-string-match (only works at last char in
//  value) and prefix match differently.
const ValueTerminator byte = 0xf5
