package go_scov

import (
	"io/ioutil"
	"strings"
	"os/exec"
	"syscall"
	"os"
	"fmt"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// "" => []  "foo" => ["foo"]
func splitWithoutEmpty(string string, delimiter rune) []string {
	return strings.FieldsFunc(string, func(c rune) bool { return c == delimiter })
}

func filter(ss []string, test func(string) bool) (filtered []string) {
	filtered = []string{}
	for _, s := range ss {
		if test(s) {
			filtered = append(filtered, s)
		}
	}
	return
}

// TODO move into a utils package ? ... how does coverage work then ?
// Run a command and stream output to stdout/err, but return an exit code
// https://stackoverflow.com/questions/10385551/get-exit-code-go
func runCommand(name string, argv []string) (exitCode int) {
	cmd := exec.Command(name, argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get
			exitCode = 1
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	return
}

// TODO use multi-string interface instead
func CovTest(argv []string) (exitCode int) {
	path := "coverage.out"
	argv = append([]string{"test"}, argv...)
	argv = append(argv, "-cover", "-coverprofile=" +path)
	exitCode = runCommand("go", argv)
	if(exitCode == 0) {
		uncovered := Uncovered(path)
		if(len(uncovered) != 0) { // TODO heading and more info maybe
			fmt.Fprintln(os.Stderr, "Uncovered lines found:")
			fmt.Fprintln(os.Stderr, strings.Join(uncovered, "\n"))
			return 1
		}
	}
	return
}

// Find the uncovered lines given a coverage file path
func Uncovered(path string) (uncoveredLines []string) {
	data, err := ioutil.ReadFile(path)
	check(err)

	lines := splitWithoutEmpty(string(data), '\n')
	if(len(lines) == 0) {
		return []string{}
	}

	// remove the initial `set: mode` line
	lines = lines[1:]

	// filter out lines that are covered (end " 0")
	lines = filter(lines, func(line string) bool { return !strings.HasSuffix(line, " 0") })

	return lines
}
