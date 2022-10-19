package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"
	//"unicode"

	//"github.com/stephen-fox/steamutil/locations"
	"github.com/stephen-fox/steamutil/shortcuts"
	//"github.com/stephen-fox/steamutil/vdf"
)

//Steam holds a Steam instance
type Steam struct {}

//NewSteam verifies a valid Steam instance with at least one user account and returns an instance of it
func NewSteam() (*Steam, error) {
	steam := &Steam{}

	rootFolder := steam.GetRootFolder()
	if rootFolder == "" {
		return nil, fmt.Errorf("unable to find Steam install")
	}

	return steam, nil
}

func (s *Steam) GetExe() string {
	exeExt := ""
	if runtime.GOOS == "windows" {
		exeExt = ".exe"
	}
	return fmt.Sprintf("%s/steam%s", s.GetRootFolder(), exeExt)
}

func (s *Steam) GetLibraryFolders() ([]string, error) {
	libraryVdf := fmt.Sprintf("%s/steamapps/libraryfolders.vdf", s.GetRootFolder())

	libraryRaw, err := ioutil.ReadFile(libraryVdf)
	if err != nil {
		return nil, err
	}
	libraryString := string(libraryRaw)

	pathRegex := regexp.MustCompile(`(?:)"path"\t\t"(.*)"(?:)`)
	pathMatches := pathRegex.FindAllStringSubmatch(libraryString, -1)

	libraryFolders := make([]string, 0)
	for _, pathMatch := range pathMatches {
		libraryFolders = append(libraryFolders, strings.ReplaceAll(pathMatch[1], `\\`, `/`))
	}

	return libraryFolders, nil
}

func (s *Steam) UserIdsToDataDirPaths() (map[string]string, error) {
	rootFolder := s.GetRootFolder()
	idsToDirs := make(map[string]string)

	infos, err := ioutil.ReadDir(rootFolder + "/userdata")
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		idsToDirs[info.Name()] = rootFolder + "/userdata/" + info.Name()
	}
	return idsToDirs, nil
}

//for now, only support one user
func (s *Steam) GetShortcutPath() (string, error) {
    userPaths, err := s.UserIdsToDataDirPaths()
	if err != nil {
		return "", err
	}
    userPath := ""
    for _, dirPath := range userPaths {
        if dirPath != "" {
            userPath = dirPath
            break
        }
    }
    if userPath == "" {
        return "", fmt.Errorf("GetShortcutPath: no user path found")
    }
    return fmt.Sprintf("%s/config/shortcuts.vdf", userPath), nil
}

func (s *Steam) GetShortcuts() ([]shortcuts.Shortcut, error) {
	shortcutPath, err := s.GetShortcutPath()
	if err != nil {
		return nil, err
	}

	shortcutFile, err := os.Open(shortcutPath)
	if err != nil {
		return nil, err
	}
	defer shortcutFile.Close()

	return shortcuts.ReadFile(shortcutFile)
}

func (s *Steam) CreateShortcut(id int, appName, exePath, startDir, iconPath, launchOptions string, tags []string) shortcuts.Shortcut {
	return shortcuts.Shortcut{
		Id: id,
		AppName: appName,
		ExePath: exePath,
		StartDir: startDir,
		IconPath: iconPath,
		LaunchOptions: launchOptions,
		Tags: tags,
	}
}

func (s *Steam) SaveShortcuts(shortcutList []shortcuts.Shortcut) error {
	shortcutPath, err := s.GetShortcutPath()
	if err != nil {
		return fmt.Errorf("Steam: SaveShortcuts(1): %v", err)
	}

	shortcutFile, err := os.OpenFile(shortcutPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("Steam: SaveShortcuts(2): %v", err)
	}
	defer shortcutFile.Close()

	if err := shortcuts.OverwriteVdfV1File(shortcutFile, shortcutList); err != nil {
		return fmt.Errorf("Steam: SaveShortcuts(3): %v", err)
	}
	return nil
}
