package main

import (
	"golang.org/x/sys/windows/registry"
)

func (s *Steam) GetRootFolder() string {
	steamKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		steamKey, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, registry.QUERY_VALUE)
		if err != nil {
			return ""
		}
	}
	defer steamKey.Close()

	installPath, _, err := steamKey.GetStringValue("InstallPath")
	if err != nil {
		return ""
	}
	return installPath
}