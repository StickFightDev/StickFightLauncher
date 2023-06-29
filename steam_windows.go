package main

import (
	"golang.org/x/sys/windows/registry"
)

func (s *Steam) GetRootFolder() string {
	steamKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		//logError("Error looking for 32-bit Steam install: %v", err)
		steamKey, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, registry.QUERY_VALUE)
		if err != nil {
			//logError("Error looking for 64-bit Steam install: %v", err)
			return "C:\\Program Files (x86)\\Steam"
		}
	}
	defer steamKey.Close()

	installPath, _, err := steamKey.GetStringValue("InstallPath")
	if err != nil {
		//logError("Error reading Steam install path key: %v", err)
		return "C:\\Program Files (x86)\\Steam"
	}
	return installPath
}