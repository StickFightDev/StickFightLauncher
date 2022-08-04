package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/JoshuaDoes/json"
)

//Status holds server statistics
type Status struct {
	Address string `json:"address"`
	Online bool `json:"online"`
	Lobbies int `json:"lobbies"`
	MaxLobbies int `json:"maxLobbies"`
	Players int `json:"playersOnline"`
}

//Manifest holds the current DLL manifest
type Manifest struct {
	BaseURL    string `json:"baseURL"`
	Assemblies []DLL  `json:"assemblies"`
}

type DLL struct {
	Filename     string `json:"filename"`
	StockSHA256  string `json:"stock"`
	ModdedSHA256 string `json:"modded"`
}

var (
	dev = false

	//Command-line flags and their defaults
	version = false
	verbosityLevel = 0
	ip = "72.9.147.58"
	port = 1337
	dllManifest = "https://raw.githubusercontent.com/StickFightDev/StickFightLauncher/dev/mod/manifest.json"
	modDll = "Assembly-CSharp.srv.dll"
	noUpdate = false
	noLauncherUpdate = false
	updated = false
	isSteam = false
	sfExe = ""

	serverStatus *Status
)

func panicLinux() {
    if runtime.GOOS == "linux" {
        logFatal("LINUX DEBUG")
    }
}

func init() {
	flag.BoolVar(&version, "version", version, "Set to display version info and exit with no operation")
	flag.IntVar(&verbosityLevel, "verbosity", verbosityLevel, "The verbosity level of debug log output (0=none, 1=debug, 2=trace)")
	flag.StringVar(&ip, "ip", ip, "The IP to connect to")
	flag.IntVar(&port, "port", port, "The port to connect to")
	flag.StringVar(&dllManifest, "dllManifest", dllManifest, "The URL of the DLL manifest to install from")
	flag.StringVar(&modDll, "modDll", modDll, "The filename to give the cached assembly")
	flag.BoolVar(&noUpdate, "noUpdate", noUpdate, "Set to only install the cached DLL, effectively offline mode")
	flag.BoolVar(&noLauncherUpdate, "noLauncherUpdate", noLauncherUpdate, "Set to disable automatic launcher updates")
	flag.BoolVar(&updated, "updated", updated, "Set to delete " + os.Args[0] + ".oudated.exe")
	flag.BoolVar(&isSteam, "steam", isSteam, "Set if launched from Steam non-game shortcut")
	flag.StringVar(&sfExe, "sfExe", sfExe, "The relative or exact path to StickFight.exe")
	flag.Parse()
}

func main() {
	if updated {
		err := os.Remove(os.Args[0] + ".outdated.exe")
		if err != nil {
			logError("unable to remove outdated launcher: %v", err)
		}
	}

	//Production updates are built with a go wrapper called govvv, but we don't want to lose our work in a dev environment
	if BuildID == "sflaunch-" + runtime.Version() {
		logPrefix("DEV", "Enabling developer mode...")
		dev = true

		logPrefix("DEV", "Disabling automatic launcher updates...")
		noLauncherUpdate = true
	}

	logPrefix("VERSION", "Stick Fight Launcher © JoshuaDoes 2022.")
	logPrefix("VERSION", "Build ID: " + BuildID)
	if version {
		return
	}

	logBlank()
	logInfo("Checking server status...")
	statusHttp, err := http.Get(fmt.Sprintf("http://%s:%d/status", ip, port))
	if err != nil {
		logError("%v", err)
		logFatal("Server cannot be reached!")
	}
	statusJSON, err := ioutil.ReadAll(statusHttp.Body)
	if err != nil {
		logFatal("%v", err)
	}
	err = json.Unmarshal(statusJSON, &serverStatus)
	if err != nil {
		logFatal("%v", err)
	}

	logInfo("Server online: %v", serverStatus.Online)
	if !serverStatus.Online {
		logFatal("Server is offline!")
	}
	logInfo("Players: %d in %d/%d lobbies", serverStatus.Players, serverStatus.Lobbies, serverStatus.MaxLobbies)

	logBlank()
	logWarning("!!! DON'T CLOSE ME !!!")
	logWarning("!! I am going to restore your game back to normal once you're finished playing !!")
	logWarning("! If you close me before the game, you'll need to re-validate or re-install it !")
	logWarning("!!! YOU HAVE BEEN WARNED !!!")
	logBlank()

	steam, err := NewSteam()
	if err != nil {
		logFatal("%v", err)
	}

	libraryFolders, err := steam.GetLibraryFolders()
	if err != nil {
		logFatal("%v", err)
	}
	logDebug("Library folders: %v", libraryFolders)

	logDebug("Searching for Stick Fight...")
	if !FindInstall(sfExe) {
		if !FindInstall("StickFight.exe") {
			found := false
			for _, libraryFolder := range libraryFolders {
				libraryPath := fmt.Sprintf("%s/steamapps/common/StickFightTheGame/StickFight.exe", libraryFolder)
				logDebug("Testing path: %s", libraryPath)
				if FindInstall(libraryPath) {
					found = true
					break
				}
			}
			if !found {
				logFatal("%v", "unable to find Stick Fight")
			}
		}
	}
	logInfo("Found Stick Fight: %s", sfExe)

	installPath := filepath.Dir(sfExe) + "/"
	managedPath := installPath + "StickFight_Data/Managed/"

	if !isSteam {
		logDebug("Getting Steam shortcuts...")
		shortcuts, err := steam.GetShortcuts()
		if err != nil {
			logWarning("%v", err)
		}

		appName := "Stick Fight: Dedicated Server"
		appPath := installPath + "StickFightLauncher.exe"
		if dev {
			appName = "Stick Fight: Dev Launcher"
			appPath = os.Args[0]
		}

		createShortcut := true
		for _, shortcut := range shortcuts {
			if shortcut.AppName == appName {
				logDebug("Shortcut already exists!")
				createShortcut = false
				//break
			}
            logDebug("Found shortcut: %v", shortcut)
		}
		
		if createShortcut {
			logInfo("Injecting Steam shortcut for %s...", appName)
			launcherArgs := fmt.Sprintf("-steam -verbosity %d -ip %s -port %d", verbosityLevel, ip, port)
			shortcut := steam.CreateShortcut(len(shortcuts),
				appName, //Use either production or dev mode naming for the shortcut
				appPath, //Use the correct path to the launcher
				installPath, //Set working directory to game directory
				sfExe, //Use Stick Fight's current icon
				launcherArgs, //Pass good enough starter args
				"favorite", "favorites", //Try to tag to favorites list
			)
			shortcuts = append(shortcuts, shortcut)

			logDebug("Saving Steam shortcuts...")
			err = steam.SaveShortcuts(shortcuts)
			if err != nil {
				logFatal("%v", err)
			}
		}

		if !dev {
			logInfo("Migrating launcher into game directory...")
			err = os.Rename(os.Args[0], installPath + "StickFightLauncher.exe")
			if err != nil {
				logError("unable to migrate launcher: %v", err)

				logDebug("Failed to migrate launcher, copying instead...")
				_, err = CopyFile(os.Args[0], installPath + "StickFightLauncher.exe")
				if err != nil {
					logFatal("unable to copy launcher: %v", err)
				}
			}
			os.Args[0] = installPath + "StickFightLauncher.exe" //Correct the os.Args slice for future use
		}
	}

	if noLauncherUpdate {
		logDebug("Skipping automatic launcher updates...")
	} else {
		logInfo("Checking for launcher updates...")
		releasesJSON := HTTPGET("https://api.github.com/repos/StickFightDev/StickFightLauncher/releases")

		releases := make([]*GitHubRelease, 0)
		err = json.Unmarshal(releasesJSON, &releases)
		if err != nil {
			logFatal("%v", err)
		}

		//Release IDs appear to be incremental, filter out the latest release
		var latest *GitHubRelease
		for _, release := range releases {
			if latest == nil {
				latest = release
				continue
			}

			if release.ID > latest.ID {
				latest = release
			}
		}

		if latest == nil {
			logError("unable to find any launcher releases")
		} else if latest.TagName != GitCommit {
			logInfo("Stick Fight Launcher (%s) is outdated, updating to (%s)...", GitCommit, latest.TagName)

			var assetExe *GitHubAsset
			for _, asset := range latest.Assets {
				if asset.ContentType == "application/x-msdownload" {
					assetExe = asset
					break
				}
			}

			if assetExe == nil {
				logError("unable to find valid application/x-msdownload asset")
			} else {
				downloadLauncher := HTTPGET(assetExe.BrowserDownloadURL)

				logDebug("Migrating outdated launcher...")
				err = os.Rename(os.Args[0], os.Args[0] + ".outdated.exe")
				if err != nil {
					logFatal("%v", err)
				}

				logDebug("Writing new launcher...")
				err = os.WriteFile(os.Args[0], downloadLauncher, 0666)
				if err != nil {
					logFatal("unable to write launcher: %v", err)
				}

				logDebug("Launching new launcher...")
				newArgs := make([]string, 0)
				if len(os.Args) > 1 {
					newArgs = os.Args[1:]
				}
				newArgs = append(newArgs, "-updated")

				launcher := exec.Command(os.Args[0], newArgs...)
				launcher.Stdout = os.Stdout
				launcher.Stderr = os.Stderr
				launcher.Stdin = os.Stdin

				err = launcher.Run()
				if err != nil {
					logFatal("Process ended with code: %v", err)
				}

				os.Exit(0)
			}
		}
	}

	installDLL := managedPath + "Assembly-CSharp.dll"
	if !PathExists(installDLL) {
		logFatal("unable to find Stick Fight assembly here: %s", installDLL)
	}
	installSHA256 := SHA256(installDLL)
	logDebug("Found Stick Fight assembly: %s (%s)", installDLL, installSHA256)

	logDebug("Backing up Stick Fight assembly...")
	backupDLL := managedPath + "Assembly-CSharp.old.dll"
	os.Rename(installDLL, backupDLL)

	logDebug("Deferring restore of Stick Fight assembly to end of main...")
	defer Restore(backupDLL, installDLL)

	serverDLL := managedPath + modDll
	dllSHA256 := SHA256(serverDLL)

	if noUpdate {
		if !PathExists(serverDLL) {
			logFatal("unable to find server assembly at path: %s", serverDLL)
		}
	} else {
		logInfo("Checking for server assembly updates...")
		jsonManifest := HTTPGET(dllManifest)
		if jsonManifest == nil {
			logFatal("unable to check for server assembly updates")
		}

		manifest := &Manifest{}
		err = json.Unmarshal(jsonManifest, manifest)
		if err != nil {
			logFatal("unable to unmarshal assembly manifest: %v", err)
		}

		found := false
		logDebug("Stock DLL: %s", installSHA256)
		for _, dll := range manifest.Assemblies {
			logDebug("Trying DLL: %v", dll)
			if dll.StockSHA256 == installSHA256 {
				if dllSHA256 != dll.ModdedSHA256 {
					logDebug("Stick Fight server assembly (%s) is outdated, updating to (%s)...", dllSHA256, dll.ModdedSHA256)
					downloadDLL := HTTPGET(manifest.BaseURL + dll.Filename)
					if downloadDLL == nil {
						logFatal("unable to download server assembly")
					}

					err := os.WriteFile(serverDLL, downloadDLL, 0666)
					if err != nil {
						logFatal("unable to write server assembly: %v", err)
					}
				}

				found = true
				break
			}
		}
		if !found {
			logFatal("unable to find current game version in assembly manifest")
		}
	}

	logInfo("Installing server assembly (%s)...", dllSHA256)
	_, err = CopyFile(serverDLL, installDLL)
	if err != nil {
		logFatal("%v", "unable to install server assembly")
	}

	logInfo("Launching Stick Fight...")
	pidTime := time.Now()
	sf := exec.Command("steam", fmt.Sprintf("steam://rungameid/674940 -address %s -port %d", ip, port))
	if runtime.GOOS == "windows" {
		sf = exec.Command("rundll32", "url.dll,FileProtocolHandler", fmt.Sprintf("steam://rungameid/674940 -address %s -port %d", ip, port))
	}
	sf.Stdout = os.Stdout
	sf.Stderr = os.Stderr
	sf.Stdin = os.Stdin
	
	err = sf.Run()
	if err != nil {
		logFatal("Failed to launch Stick Fight: %v", err)
	}

	proc, err := processFromName("StickFight.exe")
	if err != nil {
        logWarning("Waiting up to 30 seconds for Stick Fight to launch...")
        for {
            proc, err = processFromName("StickFight.exe")
            if err == nil {
                break
            }
            if time.Since(pidTime).Seconds() > 30 {
                logFatal("Unable to find Stick Fight: %v", err)
            }
        }
	}

	logInfo("Waiting for Stick Fight to exit...")
    for {
        time.Sleep(time.Second * 1)
        if running, err := proc.IsRunning(); running != true || err != nil {
            logInfo("Stick Fight ended after %d seconds with code: %v", time.Since(pidTime).Seconds(), err)
            break
        }
    }
}

func Restore(backupDLL, installDLL string) {
	logInfo("Restoring Stick Fight assembly...")
	os.Rename(backupDLL, installDLL)

	logInfo("Thank you for playing!")
	time.Sleep(time.Second * 3)
}

func FindInstall(path string) bool {
	if path == "" {
		return false
	}
	if _, err := exec.LookPath(path); err != nil {
		return false
	}

	sfExe = path
	return true
}

func PathExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func SHA256(fileName string) string {
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Sprintf("%x", fileName)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Sprintf("%x", fileName)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func HTTPGET(url string) []byte {
	res, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil
	}

	return content
}
