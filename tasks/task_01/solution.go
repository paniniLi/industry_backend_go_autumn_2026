package main

import (
	"fmt"
	"strings"
)

func greet(name string) string {
	name = strings.TrimSpace(name)

	if name == "" {
		name = "World"
	}
	return fmt.Sprintf("Hello, %s!", name)
}
