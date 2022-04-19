package main

import (
	"fmt"
)

func logInfo(msg ...interface{}) {
	logPrefix("INFO", msg...)
}
func logWarning(msg ...interface{}) {
	logPrefix("WARN", msg...)
}
func logFatal(msg ...interface{}) {
	logPrefix("FATAL", msg...)
	panic("FATAL")
}
func logDebug(msg ...interface{}) {
	if verbosityLevel >= 1 {
		logPrefix("DEBUG", msg...)
	}
}
func logTrace(msg ...interface{}) {
	if verbosityLevel >= 2 {
		logPrefix("TRACE", msg...)
	}
}
func logPrefix(prefix string, msg ...interface{}) {
	if len(msg) > 1 {
		fmt.Printf("[" + prefix + "] " + msg[0].(string) + "\n", msg[1:]...)
	} else {
		fmt.Println("[" + prefix + "] " + msg[0].(string))
	}
}
func logBlank() {
	fmt.Printf("\n")
}