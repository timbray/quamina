package quamina

// NameTracker ... TODO
type NameTracker interface {
	IsNameUsed(label []byte) bool
}
