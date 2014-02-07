/*
	GO-WARP: a Time Warp simulator written in Go
	http://pads.cs.unibo.it

	This file is part of GO-WARP.  GO-WARP is free software, you can
	redistribute it and/or modify it under the terms of the Revised BSD License.

	For more information please see the LICENSE file.

	Copyright 2014, Gabriele D'Angelo, Moreno Marzolla, Pietro Ansaloni
	Computer Science Department, University of Bologna, Italy
*/

package warp

/*
 * this is the backbone of each Logical Process, here are defined and implemented
 * all the Time Warp functions needed by the LPs
 */

import (
	"fmt"
	"os"
)

const TOOFAR = 25 // limited optimism synchronization: sets how far from the GVT a LP can go

func SimSetup(lpn int, simt Time, f func(ev *Event, l *LocalData)) {
	AllocateChans(lpn)
	GvtSetup(lpn)
	SharedSetup(lpn, simt, f)
}

/*
 * every LP must perform an initialize() operation, that creates all
 * the needed structures and variables
 */
func SimInitialize(i Pid) *LocalData {
	var data *LocalData

	data = Initialize(i)
	State[i] = LPRUNNING

	return data
}

func Simulate(data *LocalData) {

	for {

		if State[data.IndexLP] == LPSTOPPED {
			State[data.IndexLP] = LPSTOPPED
			return
		}

		receiveAll(data)

		if data.SimTime >= EndTime {
			goIdle(data)
		}

		manageEvent(data)

		if data.GvtFlag && !CheckEvaluation() {
			t := GetGvt()
			if t != ERR {
				setGvt(t, data)
			}
		}

	}
}

/*
 * creates and sends a message to the receiver that contains the event to be
 * noticed. Saves the related anti-message in sender local area
 */
func NoticeEvent(ev *Event, receiver Pid, data *LocalData) {
	var tm TimedMessage
	var msg *Message

	/* creating the message to send */
	msg = CreateMessage(data.IndexLP, receiver, *ev)

	if receiver == data.IndexLP {
		if !data.FutureEvents.Insert(ev) {
			fmt.Println("GO-WARP, ERROR: THE HEAP IS FULL -", len(data.FutureEvents))
			fmt.Println(data.IndexLP, "- GO-WARP, ERROR: EVENT NOT INSERTED!")
			os.Exit(1)
		}
	} else {
		/* sending the message */
		sendMessage(msg, data)
	}

	tm = TimedMessage{*msg, data.SimTime}

	size := Insert(tm, data.MsgSent)
	if size > TOOLARGE && State[data.IndexLP] != LPEVALGVT {
		ask4NewGvt(data)
	}
}

func receiveAll(data *LocalData) {
Loop:
	for {
		msg := Receive(data.IndexLP)

		if msg == nil {
			break Loop
		}

		manageMessage(data, msg)
	}
}

func manageMessage(data *LocalData, msg *Message) {
	switch msg.Ev.Time {
	case GVTEVAL:
		if State[data.IndexLP] != LPSTOPPED {
			evaluateLocalMin(data)
		}

	case ABORTMSG:
		State[data.IndexLP] = LPSTOPPED

	case ACK:
		gotAck(msg, data)

	default:
		sendAck(msg, data)

		if checkAntimsg(&msg.Ev, data) {
			return
		}
		if msg.Ev.Type.Flag == ANTIMSG { // anti-message
			annihilate(&(msg.Ev), data)
			return
		}
		if msg.Ev.Time < data.SimTime { // straggler message
			rollback(msg.Ev.Time, data)
		}

		/* finally we can insert the message in the heap */
		if !(data.FutureEvents).Insert(&msg.Ev) {
			fmt.Println(data.IndexLP, "- GO-WARP, ERROR: EVENT NOT INSERTED!")
			os.Exit(1)
		}
	}
}

/*
 * returns false only if it has failed managing an event (because the heap is empty)
 */
func manageEvent(data *LocalData) bool {
	var ev *Event

	data.N_PROCESSED++

	t := data.FutureEvents.GetMinTime()

	if t >= EndTime {
		goIdle(data)
		return false
	} else if t > data.SimTime {
		data.SimTime = t
	} else if t == data.SimTime {
		/* OK, DN */
	} else if t == NOTIME {
		/* heap empty */
	} else {
		fmt.Println(data.IndexLP, "- GO-WARP, ERROR: PROCESSING AN EVENT IN THE PAST!")
		os.Exit(1)
	}

	ev = data.FutureEvents.ExtractHead()
	if ev == nil {
		return false
	}

	EventManager(ev, data)

	size := Insert(*ev, data.ProcessedEvents)
	if size > TOOLARGE && State[data.IndexLP] != LPEVALGVT {
		ask4NewGvt(data)
	}

	return true
}

func rollback(t Time, data *LocalData) {
	data.SimTime = t

	el := data.ProcessedEvents.Back()
Loop:
	for el != nil {
		e := el.Value.(Event)
		el = el.Prev()
		if e.Time >= data.SimTime {
			if !data.FutureEvents.Insert(&e) {
				fmt.Println("GO-WARP, ERROR: INSERTING A PROCESSED EVENT!")
			}
			data.N_PROCESSED--
		} else {
			break Loop
		}
	}

	el = data.MsgSent.Back()
Loop1:
	for el != nil {
		mp := el.Value.(TimedMessage)
		if mp.GetTime() >= data.SimTime {
			el = el.Prev()
			anti := createAntiMessage(&mp.M)

			if mp.M.Receiver == data.IndexLP {
				annihilate(&(anti.Ev), data)
			} else {
				sendMessage(anti, data)
			}
		} else {
			break Loop1
		}
	}

	DeleteAfter(data.SimTime, data.ProcessedEvents)
	DeleteAfter(data.SimTime, data.MsgSent)

	N_rollback[data.IndexLP]++

}

func sendMessage(msg *Message, data *LocalData) {
	tm := TimedMessage{*msg, data.SimTime}
	size := Insert(tm, data.OutgoingMsg)

	if size > TOOLARGE {
		if State[data.IndexLP] != LPEVALGVT {
			ask4NewGvt(data)
		}
	}
	Send(msg)
}

func goIdle(data *LocalData) {
	if State[data.IndexLP] == LPSTOPPED {
		return
	}

	State[data.IndexLP] = LPIDLE
	term := checkAllIdle()
	if term {
		killall(data)
		State[data.IndexLP] = LPSTOPPED
	} else {
		m := BlockingReceive(data.IndexLP) // the process blocks indefinitively

		manageMessage(data, m)

		if State[data.IndexLP] != LPSTOPPED {
			State[data.IndexLP] = LPRUNNING
		}
	}
}

func createAntiMessage(msg *Message) *Message {
	var e Event
	var m Message

	e = *CreateEvent(-msg.Ev.Id, msg.Ev.Time, Info{0, 0, ANTIMSG})
	m = *CreateMessage(msg.Sender, msg.Receiver, e)

	return &m
}

func annihilate(antimsg *Event, data *LocalData) {
	done := false
	done = IsPresent(antimsg.Time, data.ProcessedEvents)

	if done && antimsg.Time <= data.SimTime {
		rollback(antimsg.Time, data)
	}

	ev := CreateEvent(-antimsg.Id, 0, Info{0, 0, 0})
	del := data.FutureEvents.DeleteExternId(ev)

	if del.Id == ERR && del.Time == ERR {
		antimsg.Time = data.SimTime // timestamping the anti-message it will be possible to rollback its reception

		Insert(*antimsg, data.AntiMsg2Annihilate)
	}
}

func checkAntimsg(ev *Event, data *LocalData) bool {
	m := CreateMessage(0, 0, *ev)
	anti := createAntiMessage(m)
	ret := Delete(anti.Ev, data.AntiMsg2Annihilate)
	return ret
}

func ask4NewGvt(data *LocalData) {
	if State[data.IndexLP] == LPSTOPPED {
		return
	}
	if CheckEvaluation() {
		return
	}
	StartEvaluation(Lpnum)

	ev := CreateEvent(0, GVTEVAL, Info{0, 0, 0})
	for i := 0; i < Lpnum; i++ {
		if State[i] != LPSTOPPED && data.IndexLP != Pid(i) {
			msg := CreateMessage(data.IndexLP, Pid(i), *ev)
			Send(msg)
		}
	}
	evaluateLocalMin(data)
}

func evaluateLocalMin(data *LocalData) {
	var mintime Time = 1000000

	State[data.IndexLP] = LPEVALGVT

	/* mintime computation and communication */
	minheap := data.FutureEvents.GetMinTime()
	minout := GetMinTime(data.OutgoingMsg)
	minack := GetMinTime(data.Acked)

	if minheap < mintime && minheap != NOTIME {
		mintime = minheap
	}
	if minout < mintime && minout != NOTIME {
		mintime = minout
	}
	if minack < mintime && minack != NOTIME {
		mintime = minack
	}

	SetLocalMin(mintime, data.IndexLP)
	data.GvtFlag = true // local min has been set

	data.Acked.Init()
}

func setGvt(gvt Time, data *LocalData) {

	if State[data.IndexLP] == LPSTOPPED {
		return
	}

	if gvt < data.Gvt {
		fmt.Println(data.IndexLP, ", GO-WARP, ERROR: THE NEW GVT VALUE IS LOWER THAN THE PREVIOUS ONE!")
		os.Exit(1)
	}
	data.GvtFlag = false
	data.Gvt = gvt

	fossilCollection(gvt, data)
}

func fossilCollection(t Time, data *LocalData) {
	DeleteBefore(t, data.ProcessedEvents)
	DeleteBefore(t, data.MsgSent)
	State[data.IndexLP] = LPRUNNING

	data.Acked.Init()
}

func sendAck(msg *Message, data *LocalData) {
	var e *Event

	if data.GvtFlag {
		e = CreateEvent(msg.Ev.Id, ACK, Info{0, 0, YOURS})
	} else {
		e = CreateEvent(msg.Ev.Id, ACK, Info{0, 0, MINE})
	}
	ack := CreateMessage(msg.Receiver, msg.Sender, *e)
	Send(ack)
}

func gotAck(msg *Message, data *LocalData) {
	var found bool = false

	el := data.OutgoingMsg.Front()
Loop:
	for el != nil {
		m := el.Value.(TimedMessage)
		if m.M.Receiver == msg.Sender && m.M.Ev.Id == msg.Ev.Id && m.M.Ev.Time != ACK {
			if msg.Ev.Type.Flag == MINE {
				data.OutgoingMsg.Remove(el)
			} else if msg.Ev.Type.Flag == YOURS {
				Insert(m, data.Acked)
				data.OutgoingMsg.Remove(el)
			} else {
				fmt.Println(data.IndexLP, "- GO-WARP, ERROR: UNKNOWN FLAG TYPE")
				os.Exit(1)
			}
			found = true
			break Loop
		}
		el = el.Next()
	}
	if !found {
		fmt.Println("GO-WARP, ERROR: WHAT ABOUT THIS ACK?")
	}

}

func checkAllIdle() bool {
	var ret bool = true
Loop:
	for i := 0; i < Lpnum; i++ {
		if State[i] != LPIDLE {
			ret = false
			break Loop
		}
	}
	return ret
}

func killall(data *LocalData) {
	ev := CreateEvent(0, ABORTMSG, Info{0, 0, 0})

	for i := 0; i < Lpnum; i++ {
		m := CreateMessage(data.IndexLP, Pid(i), *ev)
		Send(m)
	}
}
