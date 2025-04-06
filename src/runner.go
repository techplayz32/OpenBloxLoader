package installer

import (
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const targetSubDir = "RobloxPlayer"

var executableSuffix = ".exe"

func SetupRunLogging(logFilePath string) (*os.File, error) {
	// os.O_CREATE creates the file if it doesn't exist
	// os.O_WRONLY opens it for writing only
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664) // 0664 permissions
	if err != nil {
		log.Printf("Error opening log file %s: %v", logFilePath, err)
		return nil, err
	}

	log.SetOutput(logFile)

	// Ldate: date YYYY/MM/DD
	// Ltime: time HH:MM:SS
	// Lshortfile: final file name element and line number: e.g., d.go:23.
	// Llongfile: full file name and line number: e.g., /a/b/c/d.go:23.
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("--- Log Start ---")

	return logFile, nil
}

func RunRoblox() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}
	log.Printf("Searching relative to current directory: %s\n", cwd)

	robloxDirPath := filepath.Join(cwd, targetSubDir)
	log.Printf("Looking inside directory: %s\n", robloxDirPath)

	if _, err := os.Stat(robloxDirPath); os.IsNotExist(err) {
		log.Fatalf("Error: Directory '%s' does not exist in the current directory.", targetSubDir)
	} else if err != nil {
		log.Fatalf("Error accessing directory '%s': %v", targetSubDir, err)
	}

	var foundExecutablePath string

	err = filepath.WalkDir(robloxDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Warning: Error accessing path %q: %v\n", path, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		lowerName := strings.ToLower(d.Name())
		isExecutable := strings.HasSuffix(lowerName, executableSuffix) && executableSuffix != ""
		containsName := strings.Contains(lowerName, "robloxplayer")

		if isExecutable && containsName {
			log.Printf("Found potential executable: %s\n", path)
			foundExecutablePath = path

			return io.EOF
		}

		return nil
	})

	if err != nil && err != io.EOF {
		log.Fatalf("Error walking directory '%s': %v", robloxDirPath, err)
	}

	if foundExecutablePath == "" {
		log.Fatalf("Error: Could not find a suitable Roblox executable (e.g., *RobloxPlayer*.exe) in '%s'", robloxDirPath)
	}

	log.Printf("Found executable: %s\n", foundExecutablePath)

	log.Println("Attempting to launch...")

	cmd := exec.Command(foundExecutablePath)

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error starting executable '%s': %v", foundExecutablePath, err)
	}

	log.Printf("Successfully started process %d (%s).\n", cmd.Process.Pid, filepath.Base(foundExecutablePath))
	log.Println("The program will now exit, but the Roblox Player should be running.")

	err = cmd.Wait()
	if err != nil {
		log.Printf("Executable finished with error: %v", err)
	} else {
		log.Println("Executable finished successfully.")
	}
}
