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

func runContainer(image string, name string, network bool, mountdir string) int {
	var networkMode string
	if network {
		networkMode = "slirp4netns"
	} else {
		networkMode = "none"
	}

	args := []string{
		"run",
		"--interactive", 
		"--tty",
		"--rm",
		"--name", name,
		"--security-opt=label=disable",
		"--user=root",
		"--pids-limit=512",
		"--memory=1g",
		"--cpus=1",
		"--network", networkMode,
		"--workdir=/work",
		"--mount", fmt.Sprintf("type=bind,src=%s,dst=/work,rw,Z", mountdir),
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
	cleanupPtr := flag.Bool("cleanup", false, "cleanup tmpdir after exit; cannot be used with --mount")
	mountPtr := flag.String("mount", "", "mount existing host directory instead of creating a tmpdir")

	flag.Parse()

	// safety check: don't use --mount and --cleanup together
	if *mountPtr != "" && *cleanupPtr {
		fmt.Println("Error: custom mounts can not be used along with --cleanup! Exiting.")
		os.Exit(1)
	}

	instanceID := makeRandomId()
	instanceName := fmt.Sprintf("disco-%s-%s", *namePtr, instanceID)

	if debug {
		fmt.Println("image:", *imagePtr)
		fmt.Println("name:", *namePtr)
		fmt.Println("network:", *networkPtr)
		fmt.Println("instanceName:", instanceName)
		fmt.Println("mount:", *mountPtr)
	}

	var mountHostPath string
	// default case: create tmpdir
	if *mountPtr == "" {
		tmpdir_path := fmt.Sprintf("/tmp/%s", instanceName)
		tmpdir_err := os.Mkdir(tmpdir_path, 0755)
		check(tmpdir_err)
		mountHostPath = tmpdir_path
	} else {
		mountHostPath = *mountPtr
	}

	fmt.Printf("Launching container %s - your tmpdir is %s\n", instanceName, mountHostPath)
	exitCode := runContainer(*imagePtr, instanceName, *networkPtr, mountHostPath)

	if *cleanupPtr {
		fmt.Println("Removing tmpdir", mountHostPath)
		os.RemoveAll(mountHostPath)
	}

	os.Exit(exitCode)
}
