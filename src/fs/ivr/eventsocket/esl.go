// fs/ivr/eventsocket/esl

/*
*	Author : Tongxiao
*     Date : 2013-11-29
 */

package eventsocket

import (
	"bufio"
	"bytes"
	l4g "code.google.com/p/log4go"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

const readerBufSize int = 1024 << 6
const eventQueueSize int = 100
const requestTimeout int = 3000

const Header_Content_Type string = "Content-Type"
const Header_Reply_Text string = "Reply-Text"
const Header_Content_Len string = "Content-Length"

const Header_Command_Reply string = "command/reply"
const Header_Api_Response string = "api/response"
const Header_Text_Plain string = "text/event-plain"
const Header_Text_Json string = "text/event-json"
const Header_Text_Disconn string = "text/disconnect-notice"

const Value_Auth_Req string = "auth/request"
const Value_Accepted_Ok string = "+OK accepted"
const Body_Content_Ok string = "+OK"
const Body_Content_Err string = "-Err"

const Ivr_Sound_Path string = "/opt/Dev/IVR/sound/"

type ESRequest struct {
	Req_Com string
	Req_App string
	Req_Arg string
}

func newESRequest(req_com, req_app string) *ESRequest {
	req := new(ESRequest)
	req.Req_Com = req_com
	req.Req_App = req_app
	return req
}

type ESocket struct {
	conn       net.Conn
	textReader *textproto.Reader
	reader     *bufio.Reader
	err        chan error
	api        chan *Event
	cmd        chan *Event
	Running    bool
	// event      chan *Event
	Dispatcher EventDispatcher
}

type EventHeader map[string]string // key:value.

type Event struct {
	Header EventHeader
	Body   string
}

func newEvent() *Event {
	event := new(Event)
	event.Header = make(EventHeader)
	return event
}

func NewESocket(conn net.Conn, dispatcher EventDispatcher) *ESocket {
	esocket := new(ESocket)
	esocket.conn = conn
	esocket.reader = bufio.NewReaderSize(conn, readerBufSize)
	esocket.textReader = textproto.NewReader(esocket.reader)
	esocket.err = make(chan error)
	esocket.api = make(chan *Event)
	esocket.cmd = make(chan *Event)
	esocket.Dispatcher = dispatcher
	esocket.Running = true
	// esocket.event = make(chan *Event, eventQueueSize)
	return esocket
}

func (es *ESocket) Init() {
	go es.RecLoop()
}

func (es *ESocket) AnswerCall() (string, error) {
	req := newESRequest("execute", "answer")
	res, err := es.handleESRequest(req)
	if err != nil {
		l4g.Warn("AnswerCall failure for : %s", err.Error())
		return "", err
	}
	l4g.Trace("AnswerCall ok.")
	return res, nil
}

func (es *ESocket) Hangup() error {
	req := newESRequest("execute", "hangup")
	_, err := es.handleESRequest(req)
	return err
}

func (es *ESocket) Sleep(duration int) error {
	req := newESRequest("execute", "sleep")
	req.Req_Arg = strconv.Itoa(duration)
	_, err := es.handleESRequest(req)
	return err
}

func (es *ESocket) PlayAnn(annfile, param1, param2 string) error {
	req := newESRequest("execute", "playback")
	data := "{var1=" + param1 + ",var2=" + param2 + "}"
	data = data + Ivr_Sound_Path + annfile
	req.Req_Arg = data
	_, err := es.handleESRequest(req)
	return err
}

func (es *ESocket) BargeIn(barge_in bool) error {
	req := newESRequest("execute", "set")
	if barge_in {
		req.Req_Arg = "playback_terminators=any"
	} else {
		req.Req_Arg = "playback_terminators=none"
	}
	_, err := es.handleESRequest(req)
	return err
}

func (es *ESocket) StartDTMF() error {
	// time.Sleep(5 * time.Second)
	req := newESRequest("execute", "start_dtmf")
	_, err := es.handleESRequest(req)
	return err

}

func (es *ESocket) StopDTMF() error {
	req := newESRequest("execute", "stop_dtmf")
	_, err := es.handleESRequest(req)
	return err
}

func (es *ESocket) SendCmd(cmd string) (string, error) {

	if es.Running {
		l4g.Debug("Send cmd --> %s", cmd)
		fmt.Fprintf(es.conn, "%s\n\n", cmd)

		timeout := CheckTimeout(requestTimeout)
		select {
		case <-timeout:
			return "", errors.New("Timeout : " + cmd)
		case res := <-es.cmd:
			l4g.Trace("Request res : %s--%s", res.Header["Reply-Text"], res.Header["Channel-Unique-Id"])
			if strings.Contains(res.Header["Reply-Text"], "OK") {
				if len(res.Header["Channel-Unique-Id"]) > 0 {
					return res.Header["Channel-Unique-Id"], nil
				}
				return res.Header["Reply-Text"], nil
			} else {
				return "", errors.New(res.Header["Reply-Text"])
			}
		case err := <-es.err:
			return "", err
		}
	} else {
		return "", errors.New("Conn already closed")
	}

}

func (es *ESocket) handleESRequest(request *ESRequest) (string, error) {

	if es.Running {

		buf := bytes.NewBufferString("sendmsg\n")
		buf.WriteString("call-command: " + request.Req_Com + "\n")
		buf.WriteString("execute-app-name: " + request.Req_App + "\n")
		if request.Req_Arg != "" && len(request.Req_Arg) > 0 {
			buf.WriteString("execute-app-arg: " + request.Req_Arg + "\n")
		}
		buf.WriteString("event-lock: true\n")
		l4g.Trace("SendRequest ---> : \n{%s}\n", buf.String())
		fmt.Fprintf(es.conn, "%s\n", buf.String())

		timeout := CheckTimeout(requestTimeout)
		select {
		case <-timeout:
			return "", errors.New("Timeout : " + request.Req_App)
		case res := <-es.cmd:
			l4g.Trace("Request res : %s", res.Header["Reply-Text"])
			return res.Header["Reply-Text"], nil
		case err := <-es.err:
			return "", err
		}

	} else {
		return "", errors.New("Conn already closed")
	}
}

func (es *ESocket) Close() {
	close(es.api)
	close(es.cmd)
	close(es.err)
	// close(es.event)
	es.conn.Close()
	es = nil
}

func (es *ESocket) RecLoop() {
	for es.recEvent() {
	}
}

func praseHeader(msg textproto.MIMEHeader, event *Event, decode bool) {
	var err error
	for k, v := range msg {
		if decode {
			event.Header[k], err = url.QueryUnescape(v[0])
			if err != nil {
				event.Header[k] = v[0]
			}
		} else {
			event.Header[k] = v[0]
		}
	}

}

func (es *ESocket) recEvent() bool {

	msg, err := es.textReader.ReadMIMEHeader()
	if err != nil {
		fmt.Println("Error:Read header failure for", err.Error())
		return false
	}
	// l4g.Debug(">>>>> %s", msg)
	event := newEvent()

	if content := msg.Get(Header_Content_Len); content != "" {
		len, err := strconv.Atoi(content)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		b_body := make([]byte, len)
		if _, err := io.ReadFull(es.reader, b_body); err != nil {
			fmt.Println("Error:Read content failure for", err.Error())
			return false
		}
		event.Body = string(b_body)
	}
	// l4g.Debug(">>>>>>>> msgType= %s,body=%s", msg.Get(Header_Content_Type), event.Body)
	switch msg.Get(Header_Content_Type) {
	case Header_Command_Reply:
		replyText := msg.Get(Header_Reply_Text)
		if strings.Contains(replyText, "-Err") {
			es.err <- errors.New(replyText)
			return true
		}
		praseHeader(msg, event, true)
		l4g.Debug("Get cmd response : %s", event.Header)
		es.cmd <- event
	case Header_Api_Response:

	case Header_Text_Json:
		praseHeader(msg, event, true)
		tmpBody := make(map[string]interface{})
		err := json.Unmarshal([]byte(event.Body), &tmpBody)
		if err != nil {
			fmt.Println("Unmarshal json text failure for", err.Error())
			es.err <- err
			return false
		}

		for k, v := range tmpBody {
			if str, ok := v.(string); ok {
				event.Header[k] = str
			} else {
				continue
			}
		}

		if _body, _ := event.Header["_body"]; _body != "" {
			event.Body = string(_body)
			delete(event.Header, "_body")
		} else {
			event.Body = ""
		}
		es.Dispatcher.OnEvent(event)
		// es.event <- event

	case Header_Text_Plain:
	case Header_Text_Disconn:
		l4g.Debug("Disconnect-notice rec ... ")
		event.Header["Event-Name"] = "HANGUP"
		event.Header["Channel-Call-UUID"] = event.Header["Controlled-Session-Uuid"]
		es.Dispatcher.OnEvent(event)
		return false
	default:
		l4g.Warn("Unsupported event : %s", msg)
	}

	return true
}
