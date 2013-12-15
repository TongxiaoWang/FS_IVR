// FS_IVR project main.go
package main

import (
	l4g "code.google.com/p/log4go"
	"fmt"
	"fs/ivr"
	"regexp"
)

func main() {

	l4g.LoadConfiguration("log4g.xml")

	ivr.InitIVRServer(8084)
	// ivr.InitDB("tcp(172.168.2.107:3306)", "root", "root01", "ivr")
}

func exprDemo() {

	value := "147258"
	dtmfRex := regexp.MustCompile(`^147\d+`)
	if dtmfRex.MatchString(value) {
		fmt.Println("Match value ....")
	} else {
		fmt.Println("No match ...")
	}

}
