// util

/**
*  Author : Tongxiao
*  Date : 2013-10-26
 */

package eventsocket

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	NanoSec  int64 = 1
	MicroSec int64 = 1000 * NanoSec
	MilliSec int64 = 1000 * MicroSec
	Second   int64 = 1000 * MilliSec
	Minute   int64 = 60 * Second
	Hour     int64 = 60 * Minute
)

type JsonTime struct {
	time time.Time
}

func NewJsonTime() *JsonTime {
	t := &JsonTime{time.Now()}
	return t
}

func getChannelId(channel_Presence_ID string) string {
	channelPId := strings.Split(channel_Presence_ID, "@")
	if len(channelPId) > 1 {
		return channelPId[0]
	}
	return channel_Presence_ID
}

func (jTime *JsonTime) MarshalJSON() ([]byte, error) {
	return []byte(jTime.time.Format(`"` + "2006-01-02 15:04:05.000" + `"`)), nil
}

func GetDateTime() string {
	now := time.Now()
	return string(now.Format(`"` + "2006-01-02 15:04:05.000" + `"`))
}

func CheckTimeout(duration int) chan bool {
	var timeout = make(chan bool)
	go func(d int) {
		time.Sleep(time.Duration(d) * time.Millisecond)
		timeout <- true
		defer close(timeout)
	}(duration)
	return timeout
}

func CheckError(err error) {
	if err != nil {
		fmt.Println("Error :", err.Error())
		os.Exit(0)
	}
}

func GenUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid)
	if err != nil || n != len(uuid) {
		return "", err
	}
	uuid[8] = 0x80
	uuid[4] = 0x40
	return hex.EncodeToString(uuid), nil

}
