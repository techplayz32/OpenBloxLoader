package installer

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const AppSettings = `<?xml version="1.0" encoding="UTF-8"?>
<Settings>
	<ContentFolder>content</ContentFolder>
	<BaseUrl>http://www.roblox.com</BaseUrl>
</Settings>`

// SetupLogging
func SetupLogging(logFilePath string) (*os.File, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Printf("Error opening log file %s: %v", logFilePath, err)
		return nil, err
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("--- Log Start ---")
	return logFile, nil
}

// FetchVersion
func FetchVersion() string {
	resp, err := http.Get("https://clientsettingscdn.roblox.com/v2/client-version/WindowsPlayer/channel/live/")
	if err != nil {
		log.Printf("Error fetching the version: %v", err)
		return ""
	}
	defer resp.Body.Close()

	var data struct {
		Version string `json:"clientVersionUpload"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return ""
	}

	return strings.TrimSpace(data.Version)
}

// extractZip remains the same
func extractZip(zipFilePath string, destDir string) error {
	log.Printf("Extracting %s to %s...", zipFilePath, destDir)

	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return fmt.Errorf("error opening zip file %s: %w", zipFilePath, err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Printf("Error creating destination directory %s: %v", destDir, err)
		return fmt.Errorf("error creating destination directory %s: %w", destDir, err)
	}

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		cleanDestDir := filepath.Clean(destDir)
		if !strings.HasPrefix(fpath, cleanDestDir+string(os.PathSeparator)) {
			log.Printf("Warning: Illegal file path in zip %s: %s (skipped)", zipFilePath, f.Name)
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				log.Printf("Error creating directory %s from zip %s: %v", fpath, zipFilePath, err)
				return fmt.Errorf("error creating directory %s: %w", fpath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			log.Printf("Error creating parent directory for %s from zip %s: %v", fpath, zipFilePath, err)
			return fmt.Errorf("error creating parent directory for %s: %w", fpath, err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Printf("Error creating file %s from zip %s: %v", fpath, zipFilePath, err)
			return fmt.Errorf("error creating file %s: %w", fpath, err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			log.Printf("Error opening file %s within zip %s: %v", f.Name, zipFilePath, err)
			return fmt.Errorf("error opening file %s within zip: %w", f.Name, err)
		}

		_, err = io.Copy(outFile, rc)
		closeErr1 := rc.Close()
		closeErr2 := outFile.Close()

		if err != nil {
			log.Printf("Error copying content for %s from zip %s: %v", f.Name, zipFilePath, err)
			return fmt.Errorf("error copying content for %s: %w", f.Name, err)
		}
		if closeErr1 != nil {
			log.Printf("Error closing file %s within zip %s: %v", f.Name, zipFilePath, closeErr1)
			return fmt.Errorf("error closing file %s within zip: %w", f.Name, closeErr1)
		}
		if closeErr2 != nil {
			log.Printf("Error closing output file %s from zip %s: %v", fpath, zipFilePath, closeErr2)
			return fmt.Errorf("error closing output file %s: %w", fpath, closeErr2)
		}
	}

	log.Printf("Extraction completed successfully for %s into %s.", zipFilePath, destDir)
	return nil
}

// InstallRobloxPlayer
func InstallRobloxPlayer() {
	log.Println("Starting Roblox Player installation process...")

	baseExtractDir := "./RobloxPlayer"
	versionFilePath := filepath.Join(baseExtractDir, "installed_version.txt")

	// --- Fetch Latest Version ---
	fetchedVersion := FetchVersion()
	if fetchedVersion == "" {
		log.Println("Failed to fetch latest version, cannot proceed.")
		return
	}
	log.Println("Latest available version:", fetchedVersion)

	// --- Check Installed Version ---
	// Check if the version file exists
	installedVersionBytes, err := os.ReadFile(versionFilePath)
	if err == nil {
		installedVersion := strings.TrimSpace(string(installedVersionBytes))
		log.Println("Found installed version:", installedVersion)
		if installedVersion == fetchedVersion {
			log.Printf("Installed version (%s) matches latest version (%s). Installation is up-to-date.", installedVersion, fetchedVersion)
			if _, err := os.Stat(filepath.Join(baseExtractDir, "AppSettings.xml")); err == nil {
				log.Println("AppSettings.xml found. Skipping installation.")
				return
			} else {
				log.Println("Warning: Version matches, but AppSettings.xml is missing. Proceeding with installation.")
			}
			return
		} else {
			log.Printf("Installed version (%s) differs from latest version (%s). Proceeding with update.", installedVersion, fetchedVersion)
		}
	} else if os.IsNotExist(err) {
		log.Printf("No installed version file found at %s. Proceeding with fresh installation.", versionFilePath)
	} else {
		// Error reading the file (permissions, etc.)
		log.Printf("Warning: Could not read installed version file at %s: %v. Proceeding with installation.", versionFilePath, err)
	}
	// --- End Check Installed Version ---

	extractionMap := map[string]string{
		// Roblox
		"Libraries.zip":                 "",
		"shaders.zip":                   "shaders",
		"ssl.zip":                       "ssl",
		"WebView2.zip":                  "",
		"WebView2RuntimeInstaller.zip":  "WebView2RuntimeInstaller",
		"content-avatar.zip":            "content/avatar",
		"content-configs.zip":           "content/configs",
		"content-fonts.zip":             "content/fonts",
		"content-models.zip":            "content/models",
		"content-sky.zip":               "content/sky",
		"content-sounds.zip":            "content/sounds",
		"content-textures2.zip":         "content/textures",
		"content-textures3.zip":         "PlatformContent/pc/textures",
		"content-terrain.zip":           "PlatformContent/pc/terrain",
		"content-platform-fonts.zip":    "PlatformContent/pc/fonts",
		"extracontent-places.zip":       "ExtraContent/places",
		"extracontent-luapackages.zip":  "ExtraContent/LuaPackages",
		"extracontent-translations.zip": "ExtraContent/translations",
		"extracontent-models.zip":       "ExtraContent/models",
		"extracontent-textures.zip":     "ExtraContent/textures",
		"RobloxApp.zip":                 "", // Root

		// Studio
		"RobloxStudio.zip":                "", // Root
		"ApplicationConfig.zip":           "ApplicationConfig",
		"content-studio_svg_textures.zip": "content/studio_svg_textures",
		"content-qt_translations.zip":     "content/qt_translations",
		"content-api-docs.zip":            "content/api_docs",
		"extracontent-scripts.zip":        "ExtraContent/scripts",
		"BuiltInPlugins.zip":              "BuiltInPlugins",
		"BuiltInStandalonePlugins.zip":    "BuiltInStandalonePlugins",
		"LibrariesQt5.zip":                "", // Root
		"Plugins.zip":                     "Plugins",
		"Qml.zip":                         "Qml",
		"StudioFonts.zip":                 "StudioFonts",
		"redist.zip":                      "", // Root
	}

	servers := []string{
		"https://setup.rbxcdn.com/",
		"https://setup-aws.rbxcdn.com/",
		"https://setup-ak.rbxcdn.com/",
		"https://roblox-setup.cachefly.net/",
		"https://s3.amazonaws.com/setup.roblox.com/",
	}

	var manifestBody []byte
	var manifestErr error
	var successfulServer string

	log.Println("Attempting to fetch package manifest...")
	for _, server := range servers {
		manifestURL := server + fetchedVersion + "-rbxPkgManifest.txt"
		log.Println("Trying manifest URL:", manifestURL)
		resp, err := http.Get(manifestURL)
		if err != nil {
			log.Printf("Error fetching manifest from %s: %v", server, err)
			continue
		}
		bodyCloser := resp.Body
		if resp.StatusCode != http.StatusOK {
			log.Printf("Non-200 status fetching manifest from %s: %s", server, resp.Status)
			bodyCloser.Close()
			continue
		}
		manifestBody, manifestErr = io.ReadAll(bodyCloser)
		closeErr := bodyCloser.Close()
		if closeErr != nil {
			log.Printf("Warning: Error closing response body from %s: %v", server, closeErr)
		}
		if manifestErr == nil {
			successfulServer = server
			log.Println("Successfully fetched manifest from:", server)
			break
		}
		log.Printf("Error reading response body from %s: %v", server, manifestErr)
		manifestBody = nil
	}

	if manifestBody == nil {
		if manifestErr != nil {
			log.Printf("Error reading manifest from last tried server: %v", manifestErr)
		}
		log.Println("Error: Could not fetch the package manifest from any server.")
		return
	}

	bodyString := string(manifestBody)
	lines := strings.Split(strings.TrimSpace(bodyString), "\n")
	var zipFilenames []string
	for i := 1; i < len(lines); i += 4 {
		if i < len(lines) {
			filename := strings.TrimSpace(lines[i])
			if strings.HasSuffix(strings.ToLower(filename), ".zip") {
				log.Printf("Found zip file in manifest: %s", filename)
				zipFilenames = append(zipFilenames, filename)
			}
		}
	}

	if len(zipFilenames) == 0 {
		log.Println("Warning: No .zip files found in the package manifest. Cannot install components.")
		return
	}
	log.Printf("Found %d zip file(s) to download and extract.", len(zipFilenames))

	// --- Download and Extract Loop ---
	var successfulExtractions int
	var installationCompletedSuccessfully bool = true

	if err := os.MkdirAll(baseExtractDir, 0755); err != nil {
		log.Printf("Fatal: Could not create base extraction directory %s: %v", baseExtractDir, err)
		return
	} else {
		log.Printf("Ensured base extraction directory exists: %s", baseExtractDir)
	}

	for _, zipFilename := range zipFilenames {
		downloadURL := successfulServer + fetchedVersion + "-" + zipFilename
		localFilePath := filepath.Join(".", zipFilename)

		log.Printf("--- Processing: %s ---", zipFilename)
		log.Printf("Download URL: %s", downloadURL)
		log.Printf("Temporary file: %s", localFilePath)

		relativeDestDir, found := extractionMap[zipFilename]
		var finalExtractDir string
		if found {
			finalExtractDir = filepath.Join(baseExtractDir, relativeDestDir)
			log.Printf("Mapping found. Target extraction directory: %s", finalExtractDir)
		} else {
			finalExtractDir = baseExtractDir
			log.Printf("Warning: No specific mapping found for %s. Extracting to base directory: %s", zipFilename, finalExtractDir)
		}
		if err := os.MkdirAll(finalExtractDir, 0755); err != nil {
			log.Printf("Error: Could not create specific target directory %s for %s: %v. Skipping this file.", finalExtractDir, zipFilename, err)
			installationCompletedSuccessfully = false
			continue
		}

		// Download
		resp, err := http.Get(downloadURL)
		if err != nil {
			log.Printf("Error starting download for %s: %v. Skipping this file.", zipFilename, err)
			installationCompletedSuccessfully = false
			continue
		}
		bodyCloser := resp.Body
		if resp.StatusCode != http.StatusOK {
			log.Printf("Download failed for %s: Server returned status %s. Skipping this file.", zipFilename, resp.Status)
			bodyCloser.Close()
			installationCompletedSuccessfully = false
			continue
		}
		outFile, err := os.Create(localFilePath)
		if err != nil {
			log.Printf("Error creating temporary file %s: %v. Skipping this file.", localFilePath, err)
			bodyCloser.Close()
			installationCompletedSuccessfully = false
			continue
		}
		_, err = io.Copy(outFile, bodyCloser)
		closeErrOut := outFile.Close()
		closeErrResp := bodyCloser.Close()
		if err != nil {
			log.Printf("Error during download of %s: %v. Cleaning up.", zipFilename, err)
			if closeErrOut != nil {
				log.Printf("Warning: Error closing temporary file %s after download error: %v", localFilePath, closeErrOut)
			}
			_ = os.Remove(localFilePath)
			installationCompletedSuccessfully = false
			continue
		}

		if closeErrOut != nil {
			log.Printf("Warning: Error closing temporary file %s after successful download: %v", localFilePath, closeErrOut)
		}
		if closeErrResp != nil {
			log.Printf("Warning: Error closing response body for %s after successful download: %v", zipFilename, closeErrResp)
		}
		log.Printf("Download completed successfully for %s.", zipFilename)

		// Extraction
		log.Printf("Attempting extraction for %s into %s...", zipFilename, finalExtractDir)
		err = extractZip(localFilePath, finalExtractDir)
		if err != nil {
			log.Printf("Extraction failed for %s into %s: %v", zipFilename, finalExtractDir, err)
			log.Printf("Removing temporary zip file %s after failed extraction.", localFilePath)
			_ = os.Remove(localFilePath)
			installationCompletedSuccessfully = false
			continue
		}
		successfulExtractions++

		// Cleanup Temporary Zip
		log.Printf("Removing temporary zip file: %s", localFilePath)
		err = os.Remove(localFilePath)
		if err != nil {
			log.Printf("Warning: Could not remove temporary zip file %s: %v", localFilePath, err)
		}
		log.Printf("--- Finished Processing: %s ---", zipFilename)

	}

	if successfulExtractions == 0 && len(zipFilenames) > 0 {
		log.Println("No zip files were successfully extracted. Installation may be incomplete.")
		installationCompletedSuccessfully = false
	}

	// --- AppSettings Creation ---
	appSettingsCreated := false
	if len(zipFilenames) > 0 {
		appSettingsPath := filepath.Join(baseExtractDir, "AppSettings.xml")
		log.Printf("Creating/Overwriting %s...", appSettingsPath)
		if err := os.MkdirAll(filepath.Dir(appSettingsPath), 0755); err != nil {
			log.Printf("Error ensuring directory exists for AppSettings.xml: %v", err)
			installationCompletedSuccessfully = false
		} else {
			settingsFile, err := os.Create(appSettingsPath)
			if err != nil {
				log.Printf("Error creating %s: %v", appSettingsPath, err)
				installationCompletedSuccessfully = false
			} else {
				_, err = io.Copy(settingsFile, strings.NewReader(AppSettings))
				closeErr := settingsFile.Close()
				if err != nil {
					log.Printf("Error writing content to %s: %v", appSettingsPath, err)
					installationCompletedSuccessfully = false
				} else if closeErr != nil {
					log.Printf("Error closing %s after writing: %v", appSettingsPath, closeErr)
					installationCompletedSuccessfully = false
				} else {
					log.Printf("Successfully created/updated %s", appSettingsPath)
					appSettingsCreated = true
				}
			}
		}
	} else {
		log.Println("Skipping AppSettings.xml creation as no zip files were found in manifest.")
	}

	// --- Write Version File on Success ---

	if installationCompletedSuccessfully && appSettingsCreated {
		log.Printf("Writing installed version (%s) to %s", fetchedVersion, versionFilePath)
		err := os.WriteFile(versionFilePath, []byte(fetchedVersion), 0664)
		if err != nil {
			log.Printf("Error writing version file %s: %v", versionFilePath, err)
		} else {
			log.Printf("Successfully wrote version file.")
		}
	} else {
		log.Printf("Skipping writing version file due to errors or incomplete installation steps.")

		_, statErr := os.Stat(versionFilePath)
		if statErr == nil {
			log.Printf("Removing potentially outdated version file %s due to installation failure.", versionFilePath)
			_ = os.Remove(versionFilePath)
		}
	}

	log.Printf("Roblox Player setup process finished. Successfully extracted %d out of %d zip files found.", successfulExtractions, len(zipFilenames))
	if !installationCompletedSuccessfully {
		log.Println("Note: One or more steps encountered errors during the installation process.")
	}
}
