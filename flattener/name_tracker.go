package flattener

type NameTracker interface {
	IsNameUsed(label []byte) bool
}
