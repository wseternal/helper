package sys

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/wseternal/helper"
	"os"
	"strings"
)

type ProcMemInfoEntry struct {
	Name string
	OomCompactLabel string
	PID int64

	// Pss is in unit of KB
	Pss int64

	// left string not decoded
	Left string
}

func DumpProcMemInfos() ([]*ProcMemInfoEntry, error) {
	var res []*ProcMemInfoEntry
	data, err := helper.ShellCommand("dumpsys meminfo -c").CombinedOutput()
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		l := strings.TrimSpace(s.Text())
		if !strings.HasPrefix(l, "proc,") {
			continue
		}
		fields := strings.SplitN(l[5:], ",", 5)
		if len(fields) < 5 {
			fmt.Fprintf(os.Stderr, "DumpProcMemInfos: invalid proc mem info line: %s\n", l)
			continue
		}
		res = append(res, &ProcMemInfoEntry{
			Name:            fields[1],
			OomCompactLabel: fields[0],
			PID:             helper.AtoiDef(fields[2], 10, -1),
			Pss:             helper.AtoiDef(fields[3], 10, -1),
			Left:            fields[4],
		})
	}
	return res, nil
}
