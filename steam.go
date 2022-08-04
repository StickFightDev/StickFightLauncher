package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/stephen-fox/steamutil/locations"
	"github.com/stephen-fox/steamutil/shortcuts"
	//"github.com/stephen-fox/steamutil/vdf"
)

//Steam holds a Steam instance
type Steam struct {
	DV       locations.DataVerifier //Transport for getting/verifying data related file and directory locations
	SteamIDs []string               //SteamIDs of previously logged in users
}

//NewSteam verifies a valid Steam instance with at least one user account and returns an instance of it
func NewSteam() (*Steam, error) {
	if !locations.IsInstalled() {
		return nil, fmt.Errorf("steam: not installed")
	}

	dv, err := locations.NewDataVerifier()
	if err != nil {
		return nil, err
	}

	userIdsToDirs, err := dv.UserIdsToDataDirPaths()
	if err != nil {
		return nil, err
	}

	if len(userIdsToDirs) == 0 {
		return nil, fmt.Errorf("steam: no users found")
	}

	steamIds := make([]string, 0)
	for steamId := range userIdsToDirs {
		skip := false
		for _, r := range steamId {
			if !unicode.IsDigit(r) {
				skip = true
				break
			}
		}
		if !skip {
			steamIds = append(steamIds, steamId)
		}
	}

	return &Steam{DV: dv, SteamIDs: steamIds}, nil
}

func (s *Steam) GetLibraryFolders() ([]string, error) {
	libraryVdf := fmt.Sprintf("%s/steamapps/libraryfolders.vdf", s.DV.RootDirPath())

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

//for now, only support one user
func (s *Steam) GetShortcutPath() (string, error) {
    //userPath, _, err := s.DV.UserDataDirPath()
    userPaths, err := s.DV.UserIdsToDataDirPaths()
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

func (s *Steam) CreateShortcut(id int, appName, exePath, startDir, iconPath, launchOptions string, tags ...string) shortcuts.Shortcut {
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
