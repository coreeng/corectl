package env

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/shirou/gopsutil/v3/process"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type ProcessDetails struct {
	Pid  int32
	Port int32
}

const PidFileDir = "/tmp/corectl"
const NoBackgroundEnvVar = "NO_BACKGROUND"
const PortConnectMin = 30000
const PortConnectMax = 40000

func IsConnectStartup(opts EnvConnectOpts) bool {
	return !opts.Background || os.Getenv(NoBackgroundEnvVar) == ""
}

func IsConnectParent(opts EnvConnectOpts) bool {
	return opts.Background && os.Getenv(NoBackgroundEnvVar) == ""
}

func IsConnectChild(opts EnvConnectOpts) bool {
	return opts.Background && os.Getenv(NoBackgroundEnvVar) != ""
}

func SetBackgroundEnv() string {
	return fmt.Sprintf("%s=1", NoBackgroundEnvVar)
}

func WritePidFile(name string, pid int) {
	if err := os.MkdirAll(PidFileDir, 0755); err != nil {
		fmt.Printf("Unable to create directory: %v\n", err)
		return
	}
	pidFile := fmt.Sprintf("%s/%s.pid", PidFileDir, name)
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Printf("Unable to write file: %v\n", err)
		return
	}
}

func ExistingPidForConnection(name string) int {
	filename := fmt.Sprintf("%s/%s.pid", PidFileDir, name)
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		log.Printf("failed to parse PID from file %s: %v", filename, err)
		return 0
	}
	// This is a pain but os.FindProcess always returns a process even if it doesn't exist
	processes, err := process.Processes()
	if err != nil {
		fmt.Println("Error retrieving processes:", err)
		return 0
	}
	for _, proc := range processes {
		if proc.Pid == int32(pid) {
			return pid
		}
	}
	return 0
}

func KillProcess(name string, pid int32, force bool) error {
	//signal := syscall.SIGTERM
	signal := syscall.SIGINT
	if force {
		signal = syscall.SIGKILL
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}
	err = proc.Signal(signal)
	if err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}
	// Check if pid file exists and remove it
	filename := fmt.Sprintf("%s/%s.pid", PidFileDir, name)
	if _, err := os.Stat(filename); err == nil {
		if err := os.Remove(filename); err != nil {
			log.Printf("failed to remove pid file %s: %v", filename, err)
		}
	}
	return nil
}

func findEnvironmentByName(name string, environments []environment.Environment) (*environment.Environment, error) {
	for _, env := range environments {
		if name == env.Environment {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("could not find environment: %s", name)
}

func GenerateConnectPort(name string) int {
	// Generate a seed based on the environment name for reproducibility
	hash := sha256.Sum256([]byte(name))
	seed := int64(binary.BigEndian.Uint64(hash[:8]))

	src := rand.NewSource(seed)
	r := rand.New(src)

	return r.Intn(PortConnectMax-PortConnectMin) + PortConnectMin
}

// Get a structure to represent the pids from a directory keyed by filename

func GetProxyPIDs(availableEnvironments []environment.Environment) (map[string]ProcessDetails, error) {
	files, err := os.ReadDir(PidFileDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	pidMap := make(map[string]ProcessDetails)
	processes, err := process.Processes()
	if err != nil {
		log.Printf("unable to read processes: %v", err)
		return pidMap, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// string the .pid extension from the filename
		filename := strings.TrimSuffix(file.Name(), ".pid")
		// check if the filename is an environment
		if _, err := findEnvironmentByName(filename, availableEnvironments); err != nil {
			// if the filename is not an environment, skip it
			continue
		}

		content, err := os.ReadFile(fmt.Sprintf("%s/%s", PidFileDir, file.Name()))
		if err != nil {
			log.Printf("failed to read file %s: %v", file.Name(), err)
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err != nil {
			log.Printf("failed to parse PID from file %s: %v", file.Name(), err)
			continue
		}
		exists := false
		// Set value to true is the pid exists in the list of processes
		for _, proc := range processes {
			if proc.Pid == int32(pid) {
				exists = true
				connections, err := proc.Connections()
				_ = connections
				if err != nil {
					log.Printf("failed to get connections for pid %d: %v", pid, err)
					continue
				}
				if len(connections) == 0 {
					log.Printf("no connections found for pid %d", pid)
					continue
				}
				pidMap[filename] = ProcessDetails{
					Pid:  int32(pid),
					Port: int32(connections[0].Laddr.Port),
				}

			}
		}
		// If the pid does not exist, remove the file
		if !exists {
			log.Printf("removing pid file %s for pid %d", file.Name(), pid)
			if err := os.Remove(fmt.Sprintf("%s/%s", PidFileDir, file.Name())); err != nil {
				log.Printf("failed to remove pid file %s: %v", file.Name(), err)
			}
			continue
		}
	}

	return pidMap, nil
}
