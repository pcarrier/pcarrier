package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	var relatedEnv []string
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GIT_") {
			relatedEnv = append(relatedEnv, env)
		}
	}

	log.Printf("args:%v env:%v", os.Args, relatedEnv)

	bin, under, err := resolveUrl(os.Args[2])
	if err != nil {
		log.Fatalf("Could not resolve URL %s: %v", os.Args[2], err)
	}

	constructed := []string{under, under}

	cmd := exec.Command(bin, constructed...)
	log.Printf("exec %v %v", bin, constructed)

	remoteOutput, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("read stdout: %v", err)
	}
	remoteErr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("read stderr: %v", err)
	}
	go io.Copy(os.Stderr, remoteErr)
	remoteInput, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("write stdin: %v", err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("%v start: %v", bin, err)
	}

	go func() {
		buf := make([]byte, 64*1024)
		for {
			read, err := remoteOutput.Read(buf)
			if read > 0 {
				log.Printf("%v< %v", bin, strconv.Quote(string(buf[:read])))
				if _, err := os.Stdout.Write(buf[:read]); err != nil {
					log.Fatalf("Couldn't forward: %v", err)
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				log.Fatalf("Couldn't read: %v", err)
			}
		}
	}()

	go func() {
		buf := make([]byte, 64*1024)
		for {
			read, err := os.Stdin.Read(buf)
			if read > 0 {
				log.Printf("%v> %v", bin, strconv.Quote(string(buf[:read])))
				if _, err := remoteInput.Write(buf[:read]); err != nil {
					log.Fatalf("Couldn't forward: %v", err)
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				log.Fatalf("Couldn't read: %v", err)
			}
		}
	}()
	if err := cmd.Wait(); err != nil {
		log.Fatalf("%v wait: %v", bin, err)
	}
}

// resolveUrl returns the binary to use and the URL to pass it
func resolveUrl(s string) (string, string, error) {
	if !strings.HasPrefix(s, "trace://") {
		return "", "", errors.New("Not using the trace:// protocol")
	}
	u, err := url.Parse(s[8:])
	if err != nil {
		return "", "", err
	}
	name := u.Scheme
	return fmt.Sprintf("git-remote-%s", name), u.String(), nil
}
