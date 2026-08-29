package cli

import (
	"fmt"
	"os"
)

var file = os.Stderr

var status string

const clearln = "\033[2K\r"
const cred = "\033[31m"
const cgreen = "\033[32m"
const cyellow = "\033[33m"
const creset = "\033[m"

func printstatus() {
	if status == "" {
		return
	}
	file.WriteString(clearln)
	file.WriteString(cyellow)
	file.WriteString(status)
	file.WriteString(creset)
}

func print(color string, format string, args ...any) {
	file.WriteString(clearln)
	file.WriteString(color)
	file.WriteString(fmt.Sprintf(format, args...))
	file.WriteString("\n")
	file.WriteString(creset)
	printstatus()
}

func Print(format string, args ...any) {
	print("", format, args...)
}

func Error(format string, args ...any) {
	print(cred, format, args...)
}

func Status(format string, args ...any) {
	status = fmt.Sprintf(format, args...)
	printstatus()
}
