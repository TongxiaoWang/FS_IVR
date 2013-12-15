// fs/ivr

/*
*   Author : Tongxiao
*   Date : 2013-11-29
 */

package ivr

import (
	l4g "code.google.com/p/log4go"
	"fmt"
	"net"
)

const Ivr_Config_File string = "/home/Admin/Dev/Go/work/FS_IVR/src/ivr.xml"

var ivr *IVR = nil

func InitIVRServer(port int) {

	serverPort := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", serverPort)
	if err != nil {
		l4g.Error("Listening on tcp port:%d failure for %s,system will exit.", port, err.Error())
	}

	ivr = NewIVR()

	dbPersistor := NewDBPersistor("mysql", "172.16.0.154:3306", "root", "root01", "ivr")
	dbPersistor.Open()

	ivr.persistor = dbPersistor

	LoadIVRConfig(Ivr_Config_File)

	l4g.Info("IVRSever listening TCP :%d", port)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			l4g.Warn("Accept client failure for : %s", err.Error())
			break
		}
		go handleClient(clientConn)
	}

}

func handleClient(clientConn net.Conn) {

	l4g.Trace("New client :%s", clientConn.RemoteAddr().String())

	ivrChannel := NewIVRChannel(clientConn)
	ivr.channelMap[ivrChannel.ChannelName] = *ivrChannel

	defer delete(ivr.channelMap, ivrChannel.ChannelName)
	defer clientConn.Close()

	ivr.ExecuteCallFlow("root", ivrChannel)

}
