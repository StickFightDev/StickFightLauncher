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

var (
	//Command-line flags and their defaults
	version = false
	verbosityLevel = 0
	ip = "72.9.147.58"
	port = 1337
	dll = "https://raw.githubusercontent.com/StickFightDev/StickFightLauncher/dev/mod/Assembly-CSharp.srv.dll"
	dllSha256 = "https://raw.githubusercontent.com/StickFightDev/StickFightLauncher/dev/mod/SHA256"
	modDll = "Assembly-CSharp.srv.dll"
	noUpdate = false
	noLauncherUpdate = false
	updated = false
	isSteam = false
	sfExe = ""

	installDLLs = []string{
		"6058775b1416c1bf80bf3bc5cdd72ddbd55ae5482c087b884d02264dc6b0fbd1", //v25
	}
)

func init() {
	flag.BoolVar(&version, "version", version, "Set to display version info and exit with no operation")
	flag.IntVar(&verbosityLevel, "verbosity", verbosityLevel, "The verbosity level of debug log output (0=none, 1=debug, 2=trace)")
	flag.StringVar(&ip, "ip", ip, "The IP to connect to")
	flag.IntVar(&port, "port", port, "The port to connect to")
	flag.StringVar(&dll, "dll", dll, "The URL of the DLL to cache and install")
	flag.StringVar(&dllSha256, "sha256", dllSha256, "The SHA256 URL for comparing the DLL")
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
	logPrefix("VERSION", "Stick Fight Launcher © JoshuaDoes 2022.")
	logPrefix("VERSION", "Build ID: " + BuildID)
	if version {
		return
	}

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
				libraryPath := fmt.Sprintf("%s\\steamapps\\common\\StickFightTheGame\\StickFight.exe", libraryFolder)
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

	installPath := filepath.Dir(sfExe) + "\\"
	managedPath := installPath + "StickFight_Data\\Managed\\"

	if !isSteam {
		logDebug("Getting Steam shortcuts...")
		shortcuts, err := steam.GetShortcuts()
		if err != nil {
			logFatal("%v", err)
		}

		createShortcut := true
		for _, shortcut := range shortcuts {
			if shortcut.AppName == "Stick Fight: Dedicated Server" {
				logDebug("Shortcut already exists!")
				createShortcut = false
				break
			}
		}
		
		if createShortcut {
			logInfo("Injecting Steam shortcut for Stick Fight: Dedicated Server...")
			launcherArgs := fmt.Sprintf("-steam -verbosity %d", verbosityLevel)
			shortcut := steam.CreateShortcut(len(shortcuts),
				"Stick Fight: Dedicated Server",
				installPath + "StickFightLauncher.exe",
				installPath, "F:\\Games\\SteamLibrary\\steamapps\\common\\StickFightTheGame\\StickFight.exe",
				launcherArgs,
				"favorite",
			)
			shortcuts = append(shortcuts, shortcut)

			logDebug("Saving Steam shortcuts...")
			err = steam.SaveShortcuts(shortcuts)
			if err != nil {
				logFatal("%v", err)
			}
		}

		logInfo("Migrating launcher into game directory...")
		err = os.Rename(os.Args[0], installPath + "StickFightLauncher.exe")
		if err != nil {
			logFatal("%v", err)
		}
		os.Args[0] = installPath + "StickFightLauncher.exe" //Correct the os.Args slice for future use
	}

	//Production updates are built with a go wrapper called govvv, but we don't want to lose our work in a dev environment so disable launcher updates
	if BuildID == "sflaunch-" + runtime.Version() {
		logPrefix("DEV", "Disabling automatic launcher updates...")
		noLauncherUpdate = true
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

		latest := releases[0]
		if latest.TagName != GitCommit {
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
				err = os.WriteFile(os.Args[0], downloadLauncher, 0777)
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

		logFatal("%v", releases[0])
	}

	installDLL := managedPath + "Assembly-CSharp.dll"
	if !PathExists(installDLL) {
		logFatal("%v", "unable to find Stick Fight assembly")
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
		logInfo("Checking for assembly updates...")
		gitSHA256 := string(HTTPGET(dllSha256))

		if dllSHA256 != gitSHA256 {
			logDebug("Stick Fight assembly (%s) is outdated, updating to (%s)...", dllSHA256, gitSHA256)
			downloadDLL := HTTPGET(dll)
			if len(downloadDLL) == 0 {
				logFatal("%v", "unable to download server assembly")
			}

			err := os.WriteFile(serverDLL, downloadDLL, 0777)
			if err != nil {
				logFatal("unable to write server assembly: %v", err)
			}
		}
	}

	logInfo("Installing server assembly (%s)...", dllSHA256)
	_, err = CopyFile(serverDLL, installDLL)
	if err != nil {
		logFatal("%v", "unable to install server assembly")
	}

	logInfo("Launching Stick Fight...")
	sf := exec.Command("rundll32", "url.dll,FileProtocolHandler", fmt.Sprintf("steam://rungameid/674940 -address %s", ip))
	if isSteam {
		sf = exec.Command(sfExe, "-address", ip)
	}
	sf.Stdout = os.Stdout
	sf.Stderr = os.Stderr
	sf.Stdin = os.Stdin
	
	err = sf.Run()
	if err != nil {
		logFatal("Process ended with code: %v", err)
	}

	logDebug("Scanning for StickFight.exe with a 15 second timeout...")
	pid := -1
	pidTime := time.Now()
	for {
		pid, err = processID("StickFight.exe")
		if err == nil {
			break
		}
		if time.Since(pidTime).Seconds() > 15 {
			logFatal("Unable to find PID by name: %v", err)
		}
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		logFatal("Unable to find process by PID: %v", err)
	}

	logInfo("Waiting for Stick Fight to exit...")
	_, err = proc.Wait()
	if err != nil {
		logFatal("Game ended with code: %v", err)
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