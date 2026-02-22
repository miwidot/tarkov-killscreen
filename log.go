package main

import "fmt"

func debugLog(format string, args ...interface{}) {
	if debugMode {
		fmt.Printf(format, args...)
	}
}

func debugLn(args ...interface{}) {
	if debugMode {
		fmt.Println(args...)
	}
}
