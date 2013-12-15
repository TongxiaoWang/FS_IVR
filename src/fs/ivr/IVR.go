// fs/ivr  IVR

/*
*	Author : Tongxiao
*     Date : 2013-12-03
 */

package ivr

import (
	l4g "code.google.com/p/log4go"
	"errors"
	"fs/ivr/eventsocket"
	"net"
	"regexp"
	"time"
)

var noMatchErr error = errors.New("NoMatch")
var noInputErr error = errors.New("NoInput")

const IVRChannel_State_Init string = "Init"
const IVRChannel_State_Service string = "Service"
const IVRChannel_State_Hangup string = "Hangup"

const Max_DTMF_Length int = 20

var ivrNodeMap map[string]IVRNode = make(map[string]IVRNode)
var ivrPromptMap map[string]Prompt = make(map[string]Prompt)
var ivrGrammarMap map[string]Grammar = make(map[string]Grammar)

type IVR struct {
	channelMap map[string]IVRChannel
	persistor  Persistor
}

func NewIVR() *IVR {
	ivr := new(IVR)
	ivr.channelMap = make(map[string]IVRChannel)
	return ivr
}

type IVRChannel struct {
	ChannelName    string
	ChannelId      string
	Dtmf           chan string
	ChannelState   string
	ChanCreateTime time.Time
	Esocket        *eventsocket.ESocket
	DtmfValue      string
	PlaybackDone   chan bool
	NoMatchTimes   int
	NoInputTimes   int
	CallParams     map[string]string
	ActiveNode     string
	ChannelHangup  chan bool
}

func NewIVRChannel(clientConn net.Conn) *IVRChannel {
	ivrChannel := new(IVRChannel)
	ivrChannel.ChannelName = clientConn.RemoteAddr().String()
	ivrChannel.Esocket = eventsocket.NewESocket(clientConn, ivrChannel)
	ivrChannel.Dtmf = make(chan string, Max_DTMF_Length)
	ivrChannel.ChanCreateTime = time.Now()
	ivrChannel.ChannelState = IVRChannel_State_Init
	ivrChannel.PlaybackDone = make(chan bool)
	ivrChannel.CallParams = make(map[string]string)
	ivrChannel.ChannelHangup = make(chan bool)
	ivrChannel.NoInputTimes = 0
	ivrChannel.NoMatchTimes = 0

	ivrChannel.Esocket.Init()

	connId, err := ivrChannel.Esocket.SendCmd("connect\n\n")
	if err != nil {
		l4g.Error("Init IVRChannel failure for %s", err.Error())
		return nil
	}

	ivrChannel.ChannelId = connId
	l4g.Debug("Update channel[%s] connId=%s", ivrChannel.ChannelName, ivrChannel.ChannelId)
	ivrChannel.Esocket.SendCmd("event json PLAYBACK_START PLAYBACK_STOP DTMF CHANNEL_ANSWER\n\n")

	return ivrChannel
}

func (channel *IVRChannel) OnEvent(event *eventsocket.Event) {

	if event != nil {
		l4g.Debug("------------------------> New Event eventName=%s,callId=%s", event.Header["Event-Name"], event.Header["Channel-Call-UUID"])
		if event.Header["Channel-Call-UUID"] == channel.ChannelId {
			if eventName, ok := event.Header["Event-Name"]; ok {
				l4g.Trace("IVR onEvent ----->  %s", eventName)
				if "DTMF" == eventName {
					dtmf, _ := event.Header["DTMF-Digit"]
					l4g.Trace("Rec new dtmf value -> %s", dtmf)
					channel.Dtmf <- dtmf
				}
				if "PLAYBACK_STOP" == eventName {
					channel.PlaybackDone <- true
				}

				if "CHANNEL_ANSWER" == eventName {
					channel.ChannelState = IVRChannel_State_Service
					channel.CallParams["ANI"] = event.Header["Caller-Orig-Caller-ID-Number"]
					channel.CallParams["DNIS"] = event.Header["Caller-Destination-Number"]
					channel.CallParams["callId"] = event.Header["Channel-Call-UUID"]
					channel.CallParams["connId"] = event.Header["Unique-ID"]
					channel.ChannelId = event.Header["Channel-Call-UUID"]
					l4g.Trace("Show CallInfo ani=%s,dnis=%s,callId=%s,connId=%s", channel.CallParams["ANI"], channel.CallParams["DNIS"], channel.CallParams["callId"], channel.CallParams["connId"])
				}

			}
		}

		if "HANGUP" == event.Header["Event-Name"] {
			channel.Esocket.Running = false
			channel.Esocket.Close()
			channel.ChannelState = IVRChannel_State_Hangup
			channel.ChannelHangup <- true // Channel hangup.
			l4g.Info("Rec client disconnected event and close channel.")
		}
	}
}

type IVRNode interface {
	Execute(ivrChannel *IVRChannel) (string, error)
}

type AnnNode struct {
	NodeName string `xml:"name,attr"`
	Prompts  PromptEntity
	NextNode string
}

func executePrompt(prompt Prompt, ivrChannel *IVRChannel) {
	if prompt.BargeIn {
		ivrChannel.Esocket.BargeIn(true)
	} else {
		ivrChannel.Esocket.BargeIn(false)
	}
	ivrChannel.Esocket.PlayAnn(prompt.Phrase[0], prompt.PName, ivrChannel.ChannelId)
}

func (node AnnNode) Execute(ivrChannel *IVRChannel) (string, error) {

	if ivrChannel.ChannelState == IVRChannel_State_Hangup {
		return "", errors.New("channel state is invalid : hangup")
	}

	ivrChannel.ActiveNode = node.NodeName
	if len(node.Prompts.Prompt) > 0 {
		for _, promptName := range node.Prompts.Prompt {
			// Find prompt from ivrPromptMap by promptName
			if prompt, ok := ivrPromptMap[promptName]; ok {
				executePrompt(prompt, ivrChannel)
				<-ivrChannel.PlaybackDone
			} else {
				l4g.Warn("Prompt not find for promptName=%s", promptName)
			}
		}
	}

	return node.NextNode, nil
}

type MenuNode struct {
	NodeName string `xml:"name,attr"`
	Prompts  PromptEntity
	Choices  Choices
	Timeout  int
	NoInput  string
	NoMatch  string
}

func (node MenuNode) Execute(ivrChannel *IVRChannel) (string, error) {

	if ivrChannel.ChannelState == IVRChannel_State_Hangup {
		return "", errors.New("channel state is invalid : hangup")
	}

	ivrChannel.ActiveNode = node.NodeName
	// Clear dtmf channel value.
	for len(ivrChannel.Dtmf) > 0 {
		<-ivrChannel.Dtmf
	}

	if len(node.Prompts.Prompt) > 0 {
		for _, promptName := range node.Prompts.Prompt {
			// Find prompt from ivrPromptMap by promptName
			if prompt, ok := ivrPromptMap[promptName]; ok {
				executePrompt(prompt, ivrChannel)
				<-ivrChannel.PlaybackDone
			} else {
				l4g.Warn("Prompt not find for promptName=%s", promptName)
			}
		}
	}

	ivrChannel.Esocket.StartDTMF()
	defer ivrChannel.Esocket.StopDTMF()

	// Wait dtmf input.
	timeout := eventsocket.CheckTimeout(node.Timeout)
	select {
	case <-timeout:
		l4g.Warn("Timeout,no dtmf.")
		ivrChannel.NoInputTimes = ivrChannel.NoInputTimes + 1
		return node.NoInput, nil
	case dtmf := <-ivrChannel.Dtmf:

		for _, choice := range node.Choices.Choice {
			if dtmf == choice.DTMF {
				return choice.NextNode, nil
			}
		}

		l4g.Warn("No match for dtmf=%s", dtmf)
		ivrChannel.NoMatchTimes = ivrChannel.NoMatchTimes + 1
		return node.NoMatch, nil
	case <-ivrChannel.ChannelHangup:
		l4g.Trace("Channel hangup.")
		return "", errors.New("Channel hangup.")
	}

}

type GotoNode struct {
	NodeName    string `xml:"name,attr"`
	Prompts     PromptEntity
	NextNode    string
	Max_NoInput int
	Max_NoMatch int
}

func (node GotoNode) Execute(ivrChannel *IVRChannel) (string, error) {

	if ivrChannel.ChannelState == IVRChannel_State_Hangup {
		return "", errors.New("channel state is invalid : hangup")
	}

	// Clear dtmf channel value.
	for len(ivrChannel.Dtmf) > 0 {
		<-ivrChannel.Dtmf
	}

	if len(node.Prompts.Prompt) > 0 {
		for _, promptName := range node.Prompts.Prompt {
			// Find prompt from ivrPromptMap by promptName
			if prompt, ok := ivrPromptMap[promptName]; ok {
				executePrompt(prompt, ivrChannel)
				<-ivrChannel.PlaybackDone
			} else {
				l4g.Warn("Prompt not find for promptName=%s", promptName)
			}
		}
	}

	if ivrChannel.NoInputTimes >= node.Max_NoInput {
		return node.NextNode, nil
	}

	if ivrChannel.NoMatchTimes >= node.Max_NoMatch {
		return node.NextNode, nil
	}

	return ivrChannel.ActiveNode, nil

}

type RootNode struct {
	NodeName string `xml:"name,attr"`
	NextNode string
}

func (node RootNode) Execute(ivrChannel *IVRChannel) (string, error) {

	if ivrChannel.ChannelState == IVRChannel_State_Hangup {
		return "", errors.New("channel state is invalid : hangup")
	}
	ivrChannel.ActiveNode = node.NodeName
	ivrChannel.Esocket.AnswerCall()
	time.Sleep(1000 * time.Millisecond)
	return node.NextNode, nil
}

type ExitNode struct {
	NodeName string `xml:"name,attr"`
}

func (node ExitNode) Execute(ivrChannel *IVRChannel) (string, error) {

	if ivrChannel.ChannelState == IVRChannel_State_Hangup {
		return "", errors.New("channel state is invalid : hangup")
	}
	ivrChannel.ActiveNode = node.NodeName
	ivrChannel.Esocket.Hangup()
	return "", nil
}

type PromptCollectNode struct {
	NodeName string `xml:"name,attr"`
	Prompts  PromptEntity
	Grammars GrammarEntity
	NoInput  string
	NoMatch  string
	NextNode string
}

func (node PromptCollectNode) Execute(ivrChannel *IVRChannel) (string, error) {

	if ivrChannel.ChannelState == IVRChannel_State_Hangup {
		return "", errors.New("channel state is invalid : hangup")
	}

	ivrChannel.ActiveNode = node.NodeName
	// Clear dtmf channel value.
	for len(ivrChannel.Dtmf) > 0 {
		<-ivrChannel.Dtmf
	}

	if len(node.Prompts.Prompt) > 0 {
		for _, promptName := range node.Prompts.Prompt {
			// Find prompt from ivrPromptMap by promptName
			if prompt, ok := ivrPromptMap[promptName]; ok {
				executePrompt(prompt, ivrChannel)
				<-ivrChannel.PlaybackDone
			} else {
				l4g.Warn("Prompt not find for promptName=%s", promptName)
			}
		}
	}

	ivrChannel.Esocket.StartDTMF()
	defer ivrChannel.Esocket.StopDTMF()

	// Wait for dtmf input.
	if grammar, ok := ivrGrammarMap[node.Grammars.Grammar[0]]; ok {

		dtmfValue := ""
		maxDtmfLen := grammar.MaxLen

		done := false

		for !done {
			timeout := eventsocket.CheckTimeout(grammar.Timeout)
			select {
			case <-timeout:
				l4g.Warn("Timeout,no dtmf.")
				done = true
			case dtmf := <-ivrChannel.Dtmf:
				if dtmf == grammar.Terminator {
					done = true
				} else {
					dtmfValue = dtmfValue + dtmf
					if len(dtmfValue) >= maxDtmfLen {
						done = true
					}
				}
			case <-ivrChannel.ChannelHangup:
				l4g.Trace("Channel hangup.")
				return "", errors.New("Channel hangup.")
			}
		}

		l4g.Debug("Now dtmf vlaue=%s", dtmfValue)
		if len(dtmfValue) == 0 {
			// Timeout Noinput error.
			ivrChannel.NoInputTimes = ivrChannel.NoInputTimes + 1
			return node.NoInput, nil
		} else {
			dtmfRex := regexp.MustCompile(grammar.Express)
			if dtmfRex.MatchString(dtmfValue) {
				ivrChannel.DtmfValue = dtmfValue
				l4g.Trace("Collect dtmfValue=%s,nextNode=%s", ivrChannel.DtmfValue, node.NextNode)
				return node.NextNode, nil
			} else {
				// No match. NoMath error.
				ivrChannel.NoMatchTimes = ivrChannel.NoMatchTimes + 1
				return node.NoMatch, nil
			}
		}

	} else {
		l4g.Warn("Grammar not find for %s at node %s", node.Grammars.Grammar[0], node.NodeName)
		return "", errors.New("Grammar not find")
	}

}

func (ivr *IVR) ExecuteCallFlow(nodeId string, ivrChannel *IVRChannel) {
	l4g.Debug("Execute Node[%s] ... ", nodeId)
	if nodeId != "" && len(nodeId) > 0 {
		if node, ok := ivrNodeMap[nodeId]; ok {
			nodeId, err := node.Execute(ivrChannel)
			ivr.persistor.Persist(ivrChannel) // Persistor IVR data.
			if err != nil {
				l4g.Error("Execute Node failure for :%s", err.Error())

				if err == noInputErr {
					ivr.ExecuteCallFlow("NoInput", ivrChannel)
				} else if err == noMatchErr {
					ivr.ExecuteCallFlow("NoMatch", ivrChannel)
				}
				return
			}

			if nodeId != "" && len(nodeId) > 0 {
				ivr.ExecuteCallFlow(nodeId, ivrChannel)
			} else {
				l4g.Info("CallFlow end...")
			}
		} else {
			l4g.Error("NodeId not find for %s", nodeId)
		}
	}
}
