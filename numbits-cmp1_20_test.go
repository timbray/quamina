//go:build go1.20

package quamina

import "cmp"

// TODO: when Go 1.19 support is dropped, replace invocations with cmp.Compare directly.

func compare[T cmp.Ordered](x, y T) int {
	return cmp.Compare[T](x, y)
}
