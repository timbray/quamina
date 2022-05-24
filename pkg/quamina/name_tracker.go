package quamina

type NameTracker interface {
	IsNameUsed(label []byte) bool
}
