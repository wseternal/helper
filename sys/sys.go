package sys

import (
	"fmt"
	"os/user"
)

type NetStatistics struct {
	RxBytes, RxPackets, TxBytes, TxPackets, RxErrors, TxErrors uint64
	Device                                                     string `json:"-"`
}

const (
	AllZeroMAC = "00:00:00:00:00:00"
	AllZeroMACNoColon = "000000000000"
)

func GetCurrentUsername() (string, error) {
	var userName string
	user, err := user.Current()
	if err != nil {
		err = fmt.Errorf("get current user failed, %s", err)
	} else {
		userName = user.Username
	}
	return userName, err
}
