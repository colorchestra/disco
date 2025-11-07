package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
)

func makeRandomId() string {
	const chars = "0123456789abcdef"
	result := make([]byte, 4)
	for i := range result {
		result[i] = chars[rand.IntN(len(chars))]
	}
	return string(result)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func runContainer(image string, name string, network bool, tmpdir string) int {
	var networkMode string
	if network {
		networkMode = "slirp4netns"
	} else {
		networkMode = "none"
	}
	args := []string{
		"run",
		"--interactive", "--tty",
		"--rm",
		"--name", name,
		"--security-opt=label=disable",
		"--user=root",
		"--pids-limit=512",
		"--memory=1g",
		"--cpus=1",
		"--network", networkMode,
		"--workdir=/work",
		"--mount", fmt.Sprintf("type=bind,src=%s,dst=/work,rw,Z", tmpdir),
		fmt.Sprintf("%s:latest", image),
	}

	cmd := exec.Command("podman", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		// check if container returns an error because the last command in the container returned an error
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		panic(err)
	}
	return 0
}

func main() {
	debug := false
	imagePtr := flag.String("image", "debian", "image to use")
	namePtr := flag.String("name", "temp", "container name")
	networkPtr := flag.Bool("network", false, "enable network access")
	cleanupPtr := flag.Bool("cleanup", false, "cleanup tmpdir after exit")

	flag.Parse()

	random_dings := makeRandomId()
	instance_name := fmt.Sprintf("disco-%s-%s", *namePtr, random_dings)

	if debug {
		fmt.Println("image:", *imagePtr)
		fmt.Println("name:", *namePtr)
		fmt.Println("network:", *networkPtr)
		fmt.Println("instance_name:", instance_name)
	}

	tmpdir_path := fmt.Sprintf("/tmp/%s", instance_name)
	tmpdir_err := os.Mkdir(tmpdir_path, 0755)
	check(tmpdir_err)

	fmt.Printf("Launching container %s - your tmpdir is %s\n", instance_name, tmpdir_path)
	exitCode := runContainer(*imagePtr, instance_name, *networkPtr, tmpdir_path)

	if *cleanupPtr {
		os.RemoveAll(tmpdir_path)
	}

	os.Exit(exitCode)
}
