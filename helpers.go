package main

import (
	"fmt"
	"golang.org/x/sys/windows"
)

// unsafe.Sizeof(windows.ProcessEntry32{})
const processEntrySize = 568

func processID(name string) (int, error) {
	h, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil { return 0, e }
	p := windows.ProcessEntry32{Size: processEntrySize}
	for {
		e := windows.Process32Next(h, &p)
		if e != nil { return 0, e }
		if windows.UTF16ToString(p.ExeFile[:]) == name {
			return int(p.ProcessID), nil
		}
	}
	return 0, fmt.Errorf("%q not found", name)
}