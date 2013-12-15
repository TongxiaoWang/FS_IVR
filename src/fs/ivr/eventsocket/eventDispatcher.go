// EventDispatcher interface

/*
*   Author : Tongxiao
*   Date : 2013-11-07
 */
package eventsocket

type EventDispatcher interface {
	OnEvent(event *Event)
}
