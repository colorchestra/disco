package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
)

type config struct {
	Image   string
	Name    string
	Network bool
	Cleanup bool
	File    string
	Mount   string
	Debug   bool
}

func makeRandomId() string {
	const chars = "0123456789abcdef"
	result := make([]byte, 4)
	for i := range result {
		result[i] = chars[rand.IntN(len(chars))]
	}
	return string(result)
}

func copyFile(src, destDir string) error {
	// create full path for new file
	filename := filepath.Base(src)
	outPath := filepath.Join(destDir, filename)

	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	return err
}

func createWorkDirectory(mountArg, fileArg, instanceName string) string {
	// create a directory if we don't want to mount an existing one
	if mountArg == "" {
		tmpDir, err := os.MkdirTemp("", instanceName+"-")
		if err != nil {
			fmt.Println("bongus")
		}

		if fileArg != "" {
			copyFile(fileArg, tmpDir)
		}
		return tmpDir
	// or just return the one we want to mount
	} else {
		return mountArg
	}
}

func parseArgs() config {
	image := flag.String("image", "debian", "image to use")
	name := flag.String("name", "temp", "container name")
	network := flag.Bool("network", false, "enable network access")
	cleanup := flag.Bool("cleanup", false, "cleanup tmpdir after exit; cannot be used with --mount")
	file := flag.String("file", "", "file to copy inside the tmpdir")
	mount := flag.String("mount", "", "mount existing host directory instead of creating a tmpdir")
	debug := flag.Bool("debug", false, "enable debug output")

	flag.Parse()

	cfg := config{
		Image:   *image,
		Name:    *name,
		Network: *network,
		Cleanup: *cleanup,
		File:    *file,
		Mount:   *mount,
		Debug:   *debug,
	}

	// safety checks
	if cfg.Mount != "" && cfg.Cleanup {
		log.Fatal("Error: --mount cannot be used with --cleanup")
	}
	if cfg.Mount != "" && cfg.File != "" {
		log.Fatal("Error: --mount cannot be used with --file")
	}

	if cfg.Debug {
		fmt.Printf("Config: %+v\n", cfg)
	}

	return cfg
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

	cfg := parseArgs()

	instanceID := makeRandomId()
	instanceName := fmt.Sprintf("disco-%s-%s", cfg.Name, instanceID)

	mountHostPath := createWorkDirectory(cfg.Mount, cfg.File, instanceName)

	fmt.Printf("Launching container %s - your tmpdir is %s\n", instanceName, mountHostPath)
	exitCode := runContainer(cfg.Image, instanceName, cfg.Network, mountHostPath)

	if cfg.Cleanup {
		fmt.Println("Removing tmpdir", mountHostPath)
		os.RemoveAll(mountHostPath)
	}

	os.Exit(exitCode)
}
