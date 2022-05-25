package core

type NameTracker interface {
	IsNameUsed(label []byte) bool
}
