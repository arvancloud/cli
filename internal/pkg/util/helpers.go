package util

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultErrorExitCode = 1
)

// #TODO improve check error to print better

// fatal prints the message (if provided) and then exits.
func fatalErrHandler(msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}

// CheckErr prints a user friendly error to STDERR and exits with a non-zero
// exit code. Unrecognized errors will be printed with an "error: " prefix.
func CheckErr(err error) {
	checkErr(err, fatalErrHandler)
}

// checkErr formats a given error as a string and calls the passed handleErr
func checkErr(err error, handleErr func(string, int)) {
	if err == nil {
		return
	}
	fmt.Println(err)
	handleErr("", DefaultErrorExitCode)
}
