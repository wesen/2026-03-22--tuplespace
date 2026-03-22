package validation

import (
	"fmt"
	"regexp"
)

var spacePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func ValidateSpace(space string) error {
	if !spacePattern.MatchString(space) {
		return newError("invalid_space", fmt.Sprintf("space %q must match %s", space, spacePattern.String()))
	}
	return nil
}
