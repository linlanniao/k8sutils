package v2

import (
	"errors"
	"strings"
)

func RemoveSuffix(input, sep string) (string, error) {
	index := strings.LastIndex(input, sep)
	if index == -1 {
		return "", errors.New("The string does not contain " + sep)
	}
	return input[:index], nil
}
