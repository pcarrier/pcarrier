package main

import (
	"errors"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func main() {
	var relatedEnv []string
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GIT_") {
			relatedEnv = append(relatedEnv, env)
		}
	}

	repo := os.Args[2]

	host, prefix, err := resolveRef(repo)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("STARTUP\nargs:%v\nenv:%v\nhost:%v\nprefix:%v", os.Args, relatedEnv, host, prefix)

	constructed := []string{os.Args[1], host, prefix}

	kbCmd := exec.Command("git", "dx", "remote", host, prefix)
	log.Printf("exec git dx remote %v", strings.Join(constructed, " "))

	remoteOutput, err := kbCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("read stdout: %v", err)
	}
	remoteErr, err := kbCmd.StderrPipe()
	if err != nil {
		log.Fatalf("read stderr: %v", err)
	}
	remoteInput, err := kbCmd.StdinPipe()
	if err != nil {
		log.Fatalf("write stdin: %v", err)
	}

	if err := kbCmd.Start(); err != nil {
		log.Fatalf("git dx remote: %v", err)
	}

	go io.Copy(os.Stderr, remoteErr)
	go io.Copy(os.Stdout, remoteOutput)
	go io.Copy(remoteInput, os.Stdin)

	if err := kbCmd.Wait(); err != nil {
		log.Fatalf("git dx remote: %v", err)
	}
}

var (
	ErrWrongScheme = errors.New("not in dx:// schema")
)

func resolveRef(repo string) (string, string, error) {
	u, err := url.Parse(repo)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "dx" {
		return "", "", ErrWrongScheme
	}
	h := u.Host
	hostParts := strings.Split(h, ".")
	for i, j := 0, len(hostParts)-1; i < j; i, j = i+1, j-1 {
		hostParts[i], hostParts[j] = hostParts[j], hostParts[i]
	}
	pathSpec := "/refs/"
	pathSpec += strings.Join(hostParts, "/")
	pathSpec += u.Path
	return h, pathSpec, nil
}
