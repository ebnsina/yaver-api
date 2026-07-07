// Package clock provides the real Clock implementation.
package clock

import "time"

type Real struct{}

func (Real) Now() time.Time { return time.Now() }
