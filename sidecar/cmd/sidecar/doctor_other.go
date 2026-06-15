//go:build !windows

package main

import "fmt"

func runDoctor() error {
	return fmt.Errorf("--doctor is currently only available on Windows")
}
