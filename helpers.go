package main

import (
    "fmt"
    "strings"

    "github.com/shirou/gopsutil/process"
)

func processFromName(name string) (*process.Process, error) {
	processes, err := process.Processes()
    if err != nil {
        return nil, err
    }

    for _, p := range processes {
        n, err := p.Name()
        if err != nil {
            return nil, err
        }

        if strings.Contains(n, name) {
            return p, nil
        }
    }

    return nil, fmt.Errorf("processFromName: no match for %s", name)
}
