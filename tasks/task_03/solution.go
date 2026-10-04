package main

import (
	"errors"
	"fmt"
)

var ErrZero = errors.New("zero is not allowed")

func fizzBuzz(n int) (string, error) {
	switch {
	case n == 0:
		return "", ErrZero
	case n%15 == 0:
		return "FizzBuzz", nil
	case n%5 == 0:
		return "Buzz", nil
	case n%3 == 0:
		return "Fizz", nil
	default:
		return fmt.Sprintf("%d", n), nil
	}
}
