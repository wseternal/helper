package netstats

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/wseternal/helper"
	"github.com/wseternal/helper/logger"
)

type Bucket struct {
	Start, Active        int64
	RxBytes, TxBytes     int64
	RxPackets, TxPackets int64
}

type ScanState int

const (
	StateParsing ScanState = iota
	StateDevSectionFound
	StateMobileTypeFound
	StateBucketCollecting
	StateBucketCollectingEnd
)

func GetTotalDataUsageOfCurrentMonth() (usage *Bucket, err error) {
	var res []*Bucket
	if res, err = GetMobileDataUsage(); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no mobile data usage data records")
	}
	t := time.Unix(res[len(res)-1].Start/1000, 0)
	usage = &Bucket{}
	usage.Start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).Unix() * 1000
	for _, elem := range res {
		if elem.Start < usage.Start {
			logger.LogW("skip elem %+v\n", elem)
			continue
		}
		usage.TxBytes += elem.TxBytes
		usage.TxPackets += elem.TxPackets
		usage.RxBytes += elem.RxBytes
		usage.RxPackets += elem.RxPackets
		usage.Active += elem.Active
	}
	return usage, nil
}

func GetMobileDataUsage() (res []*Bucket, err error) {
	var data []byte
	data, err = helper.ShellCommand("dumpsys netstats --full").Output()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("get empty response 'dumpsys netstats'")
	}
	state := StateParsing
	r := bufio.NewScanner(bytes.NewReader(data))
	var scanned int
	for r.Scan() {
		l := strings.TrimSpace(r.Text())
		switch state {
		case StateParsing:
			if strings.HasPrefix(l, "Dev stats:") {
				state = StateDevSectionFound
			}
		case StateDevSectionFound:
			if strings.HasPrefix(l, "ident=") && strings.Contains(l, "type=MOBILE") {
				state = StateMobileTypeFound
			}
		case StateMobileTypeFound:
			if !(strings.HasPrefix(l, "bucketStart=") || strings.HasPrefix(l, "st=")) {
				break
			}
			state = StateBucketCollecting
			res = make([]*Bucket, 0)
			fallthrough
		case StateBucketCollecting:
			if !(strings.HasPrefix(l, "bucketStart=") || strings.HasPrefix(l, "st=")) {
				state = StateBucketCollectingEnd
				break
			}
			bu := &Bucket{}
			if strings.HasPrefix(l, "bucketStart=") {
				scanned, err = fmt.Sscanf(l, "bucketStart=%d activeTime=%d rxBytes=%d rxPackets=%d txBytes=%d txPackets=%d",
					&bu.Start, &bu.Active, &bu.RxBytes, &bu.RxPackets, &bu.TxBytes, &bu.TxPackets)
				if scanned != 6 {
					logger.LogW("parse bucket line %s failed, scanned: %d %s", l, scanned, err)
					break
				}
			} else {
				scanned, err = fmt.Sscanf(l, "st=%d rb=%d rp=%d tb=%d tp=%d",
					&bu.Start, &bu.RxBytes, &bu.RxPackets, &bu.TxBytes, &bu.TxPackets)
				if scanned != 5 {
					logger.LogW("parse bucket line %s failed, scanned: %d %s", l, scanned, err)
					break
				}
				bu.Start = bu.Start * 1000
				bu.Active = 3600000
			}
			res = append(res, bu)
		}
		if state == StateBucketCollectingEnd {
			break
		}
	}
	if state != StateBucketCollectingEnd {
		return nil, fmt.Errorf("parse mobile buckets failed, current state is %v", state)
	}
	return res, nil
}
