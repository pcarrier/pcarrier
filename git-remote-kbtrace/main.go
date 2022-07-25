package main

import (
	"bufio"
	"log"
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

	log.Printf("args:%v env:%v", os.Args, relatedEnv)

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

	if err := kbCmd.Start(); err != nil {
		log.Fatalf("git-remote-keybase start: %v", err)
	}

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

	if err := kbCmd.Wait(); err != nil {
		log.Fatalf("git-remote-keybase wait: %v", err)
	}
}
