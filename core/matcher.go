package core

import (
	"github.com/timbray/quamina/fields"
)

type Matcher interface {
	AddPattern(x X, pat string) error
	MatchesForFields(fields []fields.Field) ([]X, error)
	IsNameUsed(label []byte) bool
	DeletePattern(x X) error
}
