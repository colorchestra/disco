package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"os/exec"

	"go.yaml.in/yaml/v4"
)

var defaultImageName = "debian:latest"

type Config struct {
    Images []map[string]string `yaml:"images"`
}

func makeRandomId() string {
	const chars = "0123456789abcdef"
	result := make([]byte, 4)
	for i := range result {
		result[i] = chars[rand.IntN(len(chars))]
	}
	return string(result)
}

func readConfig() (Config, error) {
	defaultConfig := Config{Images: []map[string]string{ {"default": defaultImageName}}}
	data, err := os.ReadFile("disco.yaml")
    var cfg Config
	if err != nil {
        // default config if no file is found
        if errors.Is(err, fs.ErrNotExist) {
            // return defaultConfig, nil
            fmt.Println("Config file disco.yaml not found! Proceeding with defaults")
            return defaultConfig, err
        }
        // return any other error
        return Config{}, err	
	}
	
	if err := yaml.Unmarshal(data, &cfg); err != nil {
        return Config{}, err
    }
	
    return cfg, nil

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
		image,
		//fmt.Sprintf("%s:latest", image),
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

func getImageName(imageArg string, config Config) string {
	// order of precedence:
	// lookup arg image name in config file, otherwise use image name "default", otherwise use hardcoded default
	if imageArg == "" {
		// check for image named "default" in config and return full name
		for _, element := range config.Images {
			for key, schmelement := range element {
				if key == "default" {
					fmt.Printf("Using default image from config file: %s\n", schmelement)
					return schmelement
				}
			}
		}
		// if no default found in the config
		return defaultImageName
	} else {
		// check for image alias in config and return full name
		for _, element := range config.Images {
			for key, schmelement := range element {
				if key == imageArg {
					fmt.Printf("Using image from config file: %s\n", schmelement)
					return schmelement
				}
			}
		}
	}
	// if image not found in config file
	return fmt.Sprintf("%s:latest", imageArg)
}

func main() {	
	var config, _ = readConfig()
	//check(err)
	
	debug := false
	imagePtr := flag.String("image", "", "image to use")
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

	imageName := getImageName(*imagePtr, config)
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

	fmt.Printf("Launching container %s with image %s - your tmpdir is %s\n", instanceName, imageName, mountHostPath)
	exitCode := runContainer(imageName, instanceName, *networkPtr, mountHostPath)

	if *cleanupPtr {
		fmt.Println("Removing tmpdir", mountHostPath)
		os.RemoveAll(mountHostPath)
	}

	os.Exit(exitCode)
}
