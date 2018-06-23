package go_scov

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"testing"
	"bytes"
	"io"
)

func TestAwesome(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Example")
}

func noError(err error) {
	Expect(err).To(BeNil())
}

func writeFile(path string, content string) {
	err := ioutil.WriteFile(path, []byte(content), 0755)
	noError(err)
}

func withTempfile(content string, fn func(*os.File)) {
	file, err := ioutil.TempFile("", "go-scov")
	noError(err)
	defer os.Remove(file.Name())
	writeFile(file.Name(), content)
	fn(file)
}

func withTempDir(fn func(string)) {
	dir, err := ioutil.TempDir("", "go-scov")
	noError(err)
	defer os.RemoveAll(dir)
	fn(dir)
}

func withEnv(key string, value string, fn func()) {
	old := os.Getenv(key)
	os.Setenv(key, value)
	defer os.Setenv(key, old)
	fn()
}

func withFakeGo(content string, fn func()) {
	withTempDir(func(dir string){
		withEnv("PATH", dir, func(){
			writeFile(dir + "/go", "#!/bin/sh\n" + content)
			fn()
		})
	})
}

// https://stackoverflow.com/questions/10473800/in-go-how-do-i-capture-stdout-of-a-function-into-a-string
func captureStdout(fn func()) string {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	return <-outC
}

var _ = Describe("go_scov", func() {
	Describe("CovTest", func(){
		It("adds coverage to passed in arguments", func(){
			withFakeGo("touch coverage.out\necho go \"$@\"", func(){
				exitCode := -1
				output := captureStdout(func(){
					exitCode = CovTest([]string{"hello", "world"})
				})
				Expect(exitCode).To(Equal(0))
				Expect(output).To(Equal("go test hello world -cover -coverprofile=coverage.out\n"))
			})
		})

		// TODO: pass on exit code not boolean
		It("fails without adding noise", func(){
			withFakeGo("touch coverage.out\nexit 15", func(){
				exitCode := -1
				output := captureStdout(func(){
					exitCode = CovTest([]string{"hello", "world"})
				})
				Expect(exitCode).To(Equal(15))
				Expect(output).To(Equal(""))
			})
		})

		// TODO: fails without adding noise
		// TODO: does not fail when coverage is ok
		// TODO: fail when coverage is not ok
	})

	Describe("RunCommand", func(){
		It("runs the given command", func() {
			exitCode := -1
			output := captureStdout(func(){
				exitCode = runCommand("echo", []string{"123"})
			})
			Expect(exitCode).To(Equal(0))
			Expect(output).To(Equal("123\n"))
		})

		It("fails when command fails", func() {
			exitCode := -1
			output := captureStdout(func(){
				exitCode = runCommand("ls", []string{"--nope"})
			})
			Expect(exitCode).To(Equal(1))
			Expect(output).To(Equal(""))
		})
	})

	Describe("Uncovered", func() {
		It("shows nothing for empty", func() {
			withTempfile("", func(file *os.File){
				Expect(Uncovered(file.Name())).To(Equal([]string{}))
			})
		})

		It("shows uncovered", func(){
			withTempfile("mode: set\nfoo/pkg.go:1.2,3.4 1 0\n", func(file *os.File){
				Expect(Uncovered(file.Name())).To(Equal([]string{"foo/pkg.go:1.2,3.4 1 0"}))
			})
		})

		It("does not show covered", func(){
			withTempfile("mode: set\nfoo/pkg.go:1.2,3.4 1 1\n", func(file *os.File){
				Expect(Uncovered(file.Name())).To(Equal([]string{}))
			})
		})

		It("does not show covered even if coverage ends in 0", func(){
			withTempfile("mode: set\nfoo/pkg.go:1.2,3.4 1 10\n", func(file *os.File){
				Expect(Uncovered(file.Name())).To(Equal([]string{}))
			})
		})
	})
})
