package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GIT_") {
			log.Printf("env: %v", env)
		}
	}

	log.Printf("args: %v", os.Args)

	repo := os.Args[2]
	switch {
	case strings.HasPrefix(repo, "dx://dxos/"):
		repo = strings.Replace(repo, "dx://", "keybase://team/", 1)
	default:
		repo = strings.Replace(repo, "dx://", "keybase://private/", 1)
	}

	constructed := []string{os.Args[1], repo}

	kbCmd := exec.Command("git-remote-keybase", constructed...)
	log.Printf("exec git-remote-keybase %v", constructed)

	remoteOutput, err := kbCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("read stdout: %v", err)
	}
	remoteInput, err := kbCmd.StdinPipe()
	if err != nil {
		log.Fatalf("write stdin: %v", err)
	}

	kbCmd.Start()

	rin := bufio.NewWriter(remoteInput)
	rout := bufio.NewScanner(remoteOutput)

	in := bufio.NewScanner(os.Stdin)
	out := bufio.NewWriter(os.Stdout)

	go func() {
		for rout.Scan() {
			txt := rout.Text()
			log.Printf("< %v", txt)
			_, err := out.WriteString(txt + "\n")
			if err != nil {
				log.Fatalf("<: %v", err)
			}
			out.Flush()
		}
	}()

	for in.Scan() {
		txt := in.Text()
		log.Printf("> %v", txt)
		_, err := rin.WriteString(txt + "\n")
		if err != nil {
			log.Fatalf(">: %v", err)
		}
		rin.Flush()
	}
}
