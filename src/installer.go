package installer

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func FetchVersion() string {
	resp, err := http.Get("https://clientsettingscdn.roblox.com/v2/client-version/WindowsPlayer/channel/live/")
	if err != nil {
		fmt.Println("Error fetching the version:", err)
		return ""
	}
	defer resp.Body.Close()

	var data struct {
		Version string `json:"clientVersionUpload"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return ""
	}
	return data.Version
}

// function to extract the zip file
func extractZip(zipFilePath string, destDir string) error {
	fmt.Printf("Extracting %s to %s...\n", zipFilePath, destDir)

	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return fmt.Errorf("error opening zip file %s: %w", zipFilePath, err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("error creating destination directory %s: %w", destDir, err)
	}

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// --- Security Check (Zip Slip Vulnerability) ---
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}
		// --- End Security Check ---

		if f.FileInfo().IsDir() {
			fmt.Printf("  Creating directory: %s\n", fpath)
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return fmt.Errorf("error creating directory %s: %w", fpath, err)
			}
			continue
		}

		fmt.Printf("  Extracting file: %s\n", fpath)

		// create the directories for the file if they don't exist
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("error creating parent directory for %s: %w", fpath, err)
		}

		// create the destination file
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("error creating file %s: %w", fpath, err)
		}

		// open the file within the zip archive
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("error opening file %s within zip: %w", f.Name, err)
		}

		// copy the content from the zip file to the destination file
		_, err = io.Copy(outFile, rc)

		// close both files *before* checking the copy error
		closeErr1 := rc.Close()
		closeErr2 := outFile.Close()

		if err != nil { // check io.Copy error
			return fmt.Errorf("error copying content for %s: %w", f.Name, err)
		}
		if closeErr1 != nil { // check zip internal file close error
			return fmt.Errorf("error closing file %s within zip: %w", f.Name, closeErr1)
		}
		if closeErr2 != nil { // check output file close error
			return fmt.Errorf("error closing output file %s: %w", fpath, closeErr2)
		}
	}

	fmt.Println("Extraction completed successfully.")
	return nil
}

func InstallRobloxPlayer() {
	var downloadURL string
	fmt.Println("downloading Roblox...")

	servers := []string{
		"https://setup.rbxcdn.com/",
		"https://setup-aws.rbxcdn.com/",
		"https://setup-ak.rbxcdn.com/",
		"https://roblox-setup.cachefly.net/",
		"https://s3.amazonaws.com/setup.roblox.com/",
	}

	fetchedVersion := FetchVersion()
	if fetchedVersion == "" {
		fmt.Println("Failed to fetch version")
		return
	}
	fmt.Println("Found version:", fetchedVersion)

	var manifestBody []byte
	var manifestErr error
	var successfulServer string

	for _, server := range servers {
		manifestURL := server + fetchedVersion + "-rbxPkgManifest.txt"
		fmt.Println("Trying manifest URL:", manifestURL)
		resp, err := http.Get(manifestURL)
		if err != nil {
			fmt.Println("Error fetching from", server, ":", err)
			continue
		}
		bodyCloser := resp.Body
		if resp.StatusCode != http.StatusOK {
			fmt.Println("Non-200 status from", server, ":", resp.Status)
			bodyCloser.Close()
			continue
		}
		manifestBody, manifestErr = io.ReadAll(bodyCloser)
		closeErr := bodyCloser.Close()
		if closeErr != nil {
			fmt.Println("Error closing response body:", closeErr)
		}
		if manifestErr == nil {
			successfulServer = server
			fmt.Println("Successfully fetched manifest from:", server)
			break
		}
		fmt.Println("Error reading response from", server, ":", manifestErr)
		manifestBody = nil
	}

	if manifestBody == nil {
		if manifestErr != nil {
			fmt.Println("Error reading manifest from last tried server:", manifestErr)
		}
		fmt.Println("Error fetching the manifest from all servers")
		return
	}

	bodyString := string(manifestBody)
	// fmt.Println("Manifest content:", bodyString)

	var filenames []string
	lines := strings.Split(strings.TrimSpace(bodyString), "\n")
	for i := 1; i < len(lines); i += 4 {
		if i < len(lines) {
			filenames = append(filenames, strings.TrimSpace(lines[i]))
		}
	}

	var targetFilename string
	for _, fname := range filenames {
		if strings.HasSuffix(fname, ".zip") {
			targetFilename = fname
			break
		}
		// todo: change the RobloxPlayerInstaller to something else
		if strings.Contains(fname, "RobloxPlayerInstaller") && strings.HasSuffix(fname, ".exe") {
			targetFilename = fname
		}
	}

	if targetFilename == "" {
		fmt.Println("Could not identify a suitable installer (.zip or .exe) in the manifest.")
		return
	}
	fmt.Printf("Identified target file in manifest: %s\n", targetFilename)

	downloadURL = successfulServer + fetchedVersion + "-" + targetFilename
	fmt.Printf("Constructed download URL: %s\n", downloadURL)

	// --- Download Section ---
	fmt.Printf("Attempting to download %s...\n", targetFilename)
	resp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Printf("Error starting download: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Download failed: Server returned status %s\n", resp.Status)
		return
	}

	localFilePath := filepath.Base(targetFilename)
	fmt.Printf("Saving file as: %s\n", localFilePath)

	outFile, err := os.Create(localFilePath)
	if err != nil {
		fmt.Printf("Error creating file %s: %v\n", localFilePath, err)
		return
	}
	defer outFile.Close()

	fmt.Println("Download starting...")
	bytesCopied, err := io.Copy(outFile, resp.Body)

	if err != nil {
		fmt.Printf("Error during download: %v\n", err)
		_ = os.Remove(localFilePath)
		return
	}

	fmt.Printf("Download completed successfully! %d bytes written to %s\n", bytesCopied, localFilePath)

	// --- Extraction Section ---
	if strings.HasSuffix(strings.ToLower(localFilePath), ".zip") {
		extractDir := "./RobloxPlayer"

		err = extractZip(localFilePath, extractDir)
		if err != nil {
			fmt.Printf("Extraction failed: %v\n", err)
			// todo: continue from here
			return
		}

		fmt.Printf("Successfully extracted %s to %s\n", localFilePath, extractDir)

		fmt.Printf("Removing temporary zip file: %s\n", localFilePath)
		err = os.Remove(localFilePath)
		if err != nil {
			fmt.Printf("Warning: Could not remove zip file %s: %v\n", localFilePath, err)
		}

	} else {
		fmt.Printf("Downloaded file %s is not a zip file, skipping extraction.\n", localFilePath)
		fmt.Println("Installation might require running the downloaded executable manually.")
	}

	fmt.Println("\nRoblox Player setup process finished.")

}
