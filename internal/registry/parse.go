package registry

import (
	"errors"
	"strings"
	"unicode"
)

func ParsePhoneNumber(input string) (PhoneNumber, error) {
	if input == "" {
		return "", errors.New("empty phone number")
	}

	if !strings.HasPrefix(input, "+") {
		return "", errors.New("phone number must be in international format (+...)")
	}

	for _, r := range input[1:] {
		if !unicode.IsDigit(r) {
			return "", errors.New("phone number must contain digits only")
		}
	}

	return PhoneNumber(input), nil
}

func IsPhoneNumber(input string) bool {
	_, err := ParsePhoneNumber(input)
	return err == nil
}
