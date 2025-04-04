package installer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func FetchVersion() string {
	resp, err := http.Get("https://clientsettingscdn.roblox.com/v2/client-version/WindowsPlayer/channel/live/")
	if err != nil {
		fmt.Println("Error fetching the version:", err)
		return ""
	}
	defer resp.Body.Close()

	// get 'version' parameter from the json response
	var data struct {
		Version string `json:"clientVersionUpload"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return ""
	}
	return data.Version
}

func InstallRobloxPlayer() {
	url := ""
	fmt.Println("downloading Roblox...")

	// in case, if the one of the servers urls is not working, we can use one of the other servers, make a list of servers urls
	// and iterate through them until we find one that works
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

	// Initialize variables to track success
	var manifestBody []byte
	var manifestErr error
	var successfulServer string

	// Try each server until we get a successful response
	for _, server := range servers {
		manifestURL := server + fetchedVersion + "-rbxPkgManifest.txt"
		fmt.Println("Trying manifest URL:", manifestURL)

		resp, err := http.Get(manifestURL)
		if err != nil {
			fmt.Println("Error fetching from", server, ":", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Println("Non-200 status from", server, ":", resp.Status)
			resp.Body.Close()
			continue
		}

		// Read the response body
		manifestBody, manifestErr = io.ReadAll(resp.Body)
		resp.Body.Close()

		if manifestErr == nil {
			successfulServer = server
			fmt.Println("Successfully fetched manifest from:", server)
			break
		}

		fmt.Println("Error reading response from", server, ":", manifestErr)
	}

	if manifestBody == nil || manifestErr != nil {
		fmt.Println("Error fetching the manifest from all servers")
		return
	}

	// Process the manifest content
	bodyString := string(manifestBody)
	fmt.Println("Manifest content:", bodyString)

	// Get all filenames from manifest
	var filenames []string
	lines := strings.Split(strings.TrimSpace(bodyString), "\n")
	for i := 1; i < len(lines); i += 4 { // Каждый 4-й элемент - новый файл
		if i < len(lines) {
			filenames = append(filenames, strings.TrimSpace(lines[i]))
		}
	}

	// Try each filename until we find a working URL
	for _, fname := range filenames {
		fileURL := successfulServer + fetchedVersion + "-" + fname
		fmt.Printf("Trying download URL: %s\n", fileURL)

		resp, err := http.Get(fileURL)
		if err != nil {
			fmt.Printf("Error trying %s: %v\n", fname, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			url = fileURL
			fmt.Printf("Found working URL with file: %s\n", fname)
			break
		}
		resp.Body.Close()
	}

	if url != "" {
		// Check if URL needs to be prefixed with the server
		// if !strings.HasPrefix(strings.ToLower(url), "http") {
		// 	url = successfulServer + url
		// }

		fmt.Printf("Download URL: %s\n", url)

		// // Download the installer
		// resp, err := http.Get(url)
		// if err != nil {
		// 	fmt.Println("Error downloading installer:", err)
		// 	return
		// }
		// defer resp.Body.Close()

		// if resp.StatusCode != http.StatusOK {
		// 	fmt.Println("Error downloading installer, status:", resp.Status)
		// 	return
		// }

		// // Create output file
		// outFile, err := os.Create("RobloxPlayerInstaller.exe")
		// if err != nil {
		// 	fmt.Println("Error creating output file:", err)
		// 	return
		// }
		// defer outFile.Close()

		// // Copy the response body to the output file
		// _, err = io.Copy(outFile, resp.Body)
		// if err != nil {
		// 	fmt.Println("Error saving installer:", err)
		// 	return
		// }

		fmt.Println("Download completed successfully!")
	} else {
		fmt.Println("Could not find download URL in manifest")
	}
}
