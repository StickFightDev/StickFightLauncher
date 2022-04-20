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
	"time"
)

var (
	//Command-line flags and their defaults
	verbosityLevel = 0
	ip = "72.9.147.58"
	port = 1337
	dll = "https://raw.githubusercontent.com/StickFightDev/StickFightLauncher/dev/mod/Assembly-CSharp.srv.dll"
	dllSha256 = "https://raw.githubusercontent.com/StickFightDev/StickFightLauncher/dev/mod/SHA256"
	modDll = "Assembly-CSharp.srv.dll"
	isSteam = false
	sfExe = ""

	installDLLs = []string{
		"6058775b1416c1bf80bf3bc5cdd72ddbd55ae5482c087b884d02264dc6b0fbd1", //v25
	}
)

func init() {
	flag.IntVar(&verbosityLevel, "verbosity", verbosityLevel, "The verbosity level of debug log output")
	flag.StringVar(&ip, "ip", ip, "The IP to connect to")
	flag.IntVar(&port, "port", port, "The port to connect to")
	flag.StringVar(&dll, "dll", dll, "The URL of the DLL to use")
	flag.StringVar(&dllSha256, "sha256", dllSha256, "The SHA256 of the DLL to compare against")
	flag.StringVar(&modDll, "modDll", modDll, "The filename to give the cached assembly")
	flag.BoolVar(&isSteam, "steam", isSteam, "Set to true if launched from Steam non-game shortcut")
	flag.StringVar(&sfExe, "sfExe", sfExe, "The relative or exact path to StickFight.exe")
	flag.Parse()
}

func main() {
	logWarning("!!! DON'T CLOSE ME !!!")
	logWarning("!! I am going to restore your game back to normal once you're finished playing !!")
	logWarning("! If you close me before the game, you'll need to re-validate or re-install it !")
	logWarning("!!! YOU HAVE BEEN WARNED !!!")
	logBlank()

	logInfo("Searching for Stick Fight...")
	if !FindInstall(sfExe) {
		if !FindInstall("StickFight.exe") {
			if !FindInstall("C:\\Program Files (x86)\\Steam\\steamapps\\common\\StickFightTheGame\\StickFight.exe") {
				logFatal("unable to find Stick Fight")
			}
		}
	}
	logInfo("Found Stick Fight: %s", sfExe)

	installPath := filepath.Dir(sfExe)
	managedPath := installPath + "\\StickFight_Data\\Managed\\"

	installDLL := managedPath + "Assembly-CSharp.dll"
	if !PathExists(installDLL) {
		logFatal("unable to find Stick Fight assembly")
	}
	installSHA256 := SHA256(installDLL)
	logInfo("Found Stick Fight assembly: %s (%s)", installDLL, installSHA256)

	logInfo("Backing up Stick Fight assembly...")
	backupDLL := managedPath + "Assembly-CSharp.old.dll"
	os.Rename(installDLL, backupDLL)

	logDebug("Deferring restore of Stick Fight assembly to end of main...")
	defer Restore(backupDLL, installDLL)

	serverDLL := managedPath + modDll
	dllSHA256 := SHA256(serverDLL)
	gitSHA256 := string(HTTPGET(dllSha256))

	if !PathExists(serverDLL) || dllSHA256 != gitSHA256 {
		logInfo("Stick Fight assembly (%s) is outdated, updating to (%s)...", dllSHA256, gitSHA256)
		downloadDLL := HTTPGET(dll)
		if len(downloadDLL) == 0 {
			logFatal("unable to download server assembly")
		}

		err := os.WriteFile(serverDLL, downloadDLL, 0777)
		if err != nil {
			logFatal("unable to write server assembly")
		}
	}

	logInfo("Installing server assembly...")
	_, err := CopyFile(serverDLL, installDLL)
	if err != nil {
		logFatal("unable to install server assembly")
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
		logFatal("Process ended with code: ", err)
	}

	pid := -1
	pidTime := time.Now()
	for {
		pid, err = processID("StickFight.exe")
		if err == nil {
			break
		}
		if time.Since(pidTime).Seconds() > 5 {
			logFatal("Unable to find PID by name: ", err)
		}
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		logFatal("Unable to find process by PID: ", err)
	}

	logInfo("Waiting for game to exit...")
	_, err = proc.Wait()
	if err != nil {
		logFatal("Game ended with code: ", err)
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