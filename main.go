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

	"github.com/JoshuaDoes/logger"
)

var (
	//Command-line flags and their defaults
	ip = "72.9.147.58"
	port = 1337
	dll = "https://raw.githubusercontent.com/StickFightDev/StickFightLauncher/dev/mod/Assembly-CSharp.srv.dll"
	dllSha256 = "https://raw.githubusercontent.com/StickFightDev/StickFightLauncher/dev/mod/SHA256"
	verbosityLevel = 0
	installExe = ""
	modDll = "Assembly-CSharp.srv.dll"

	//Things to be used by the launcher
	log *logger.Logger

	installDLLs = []string{
		"6058775b1416c1bf80bf3bc5cdd72ddbd55ae5482c087b884d02264dc6b0fbd1", //v25
	}
)

func init() {
	flag.StringVar(&ip, "ip", ip, "The IP to connect to")
	flag.IntVar(&port, "port", port, "The port to connect to")
	flag.StringVar(&dll, "dll", dll, "The URL of the DLL to use")
	flag.StringVar(&dllSha256, "sha256", dllSha256, "The SHA256 of the DLL to compare against")
	flag.StringVar(&installExe, "installExe", installExe, "The relative or exact path to StickFight.exe")
	flag.StringVar(&modDll, "modDll", modDll, "The filename to give the cached assembly")
	flag.IntVar(&verbosityLevel, "verbosity", verbosityLevel, "The verbosity level of debug log output")
	flag.Parse()

	log = logger.NewLogger("sf:launch", verbosityLevel)
}

func main() {
	//panic(SHA256("mod/Assembly-CSharp.srv.dll"))

	log.Info("Searching for Stick Fight...")
	if !FindInstall(installExe) {
		if !FindInstall("StickFight.exe") {
			if !FindInstall("C:\\Program Files (x86)\\Steam\\steamapps\\common\\StickFightTheGame\\StickFight.exe") {
				log.Fatal("unable to find Stick Fight")
			}
		}
	}
	log.Info("Found Stick Fight: ", installExe)

	installPath := filepath.Dir(installExe)
	managedPath := installPath + "\\StickFight_Data\\Managed\\"

	installDLL := managedPath + "Assembly-CSharp.dll"
	if !PathExists(installDLL) {
		log.Fatal("unable to find Stick Fight assembly")
	}
	installSHA256 := SHA256(installDLL)
	log.Info("Found Stick Fight assembly: ", installDLL, " (" + installSHA256 + ")")

	log.Info("Backing up Stick Fight assembly...")
	backupDLL := managedPath + "Assembly-CSharp.old.dll"
	os.Rename(installDLL, backupDLL)

	log.Debug("Deferring restore of Stick Fight assembly to end of main...")
	defer Restore(backupDLL, installDLL)

	serverDLL := managedPath + modDll
	dllSHA256 := SHA256(serverDLL)
	gitSHA256 := string(HTTPGET(dllSha256))

	if !PathExists(serverDLL) || dllSHA256 != gitSHA256 {
		log.Info("Stick Fight assembly (" + dllSHA256 + ") is outdated, updating to (" + gitSHA256 + ")...")
		downloadDLL := HTTPGET(dll)
		if len(downloadDLL) == 0 {
			log.Fatal("unable to download server assembly")
		}

		err := os.WriteFile(serverDLL, downloadDLL, 0777)
		if err != nil {
			log.Fatal("unable to write server assembly")
		}
	}

	log.Info("Installing server assembly...")
	_, err := CopyFile(serverDLL, installDLL)
	if err != nil {
		log.Fatal("unable to install server assembly")
	}

	log.Info("Launching Stick Fight...")
	sf := exec.Command(installExe,
		"-address", ip, //"-address", fmt.Sprintf("%s:%d", ip, port),
	)
	sf.Stdout = os.Stdout
	sf.Stderr = os.Stderr
	sf.Stdin = os.Stdin
	log.Info("Process ended with code: ", sf.Run())
}

func Restore(backupDLL, installDLL string) {
	log.Info("Restoring Stick Fight assembly...")
	os.Rename(backupDLL, installDLL)
}

func FindInstall(path string) bool {
	if path == "" {
		return false
	}
	if _, err := exec.LookPath(path); err != nil {
		return false
	}

	installExe = path
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
		panic(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		panic(err)
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