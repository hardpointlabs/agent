package common

import (
	"bufio"
	"os"
	"strings"
)

// Checks whether we're running in a container-like environment. This could be:
// - docker, podman, or any kind of container runtime
// returns true if we appear to be containerised, false otherwise
func IsContainer() bool {
	//  docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// podman
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}

	file, err := os.Open("/proc/1/cgroup")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if containsContainerMarker(line) {
			return true
		}
	}
	return false
}

func containsContainerMarker(s string) bool {
	markers := []string{"docker", "lxc", "containerd", "kubepods"}
	for _, marker := range markers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}
