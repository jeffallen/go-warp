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
 * This package contains all the default variables and structures,
 * each LP must import this package and initialize these variables
 *
 */

import (
	list "container/list"
	"fmt"
	"os"
)

type LocalData struct {
	N_PROCESSED        int
	SimTime            Time
	Gvt                Time
	IndexLP            Pid
	FutureEvents       EventHeap
	ProcessedEvents    *list.List
	MsgSent            *list.List
	AntiMsg2Annihilate *list.List
	OutgoingMsg        *list.List
	Acked              *list.List
	Pending            bool
	GvtFlag            bool
}

/*
 * every LP must perform an initialize() operation, that creates
 * the needed structures and variables
 */
func Initialize(i Pid) *LocalData {
	var d LocalData = *new(LocalData)

	/* initialize all LP variables */
	d.N_PROCESSED = 0
	d.IndexLP = i
	d.SimTime = 0
	d.Gvt = 0
	d.Pending = true
	d.GvtFlag = false
	d.FutureEvents = InitializeHeap()
	d.ProcessedEvents = NewList()
	d.MsgSent = NewList()
	d.AntiMsg2Annihilate = NewList()
	d.OutgoingMsg = NewList()
	d.Acked = NewList()

	return &d
}

func (l *LocalData) NewEvent(ev *Event) {
	if !l.FutureEvents.Insert(ev) {
		fmt.Println("GO-WARP, ERROR: EVENT NOT INSERTED")
		l.FutureEvents.Print()
		os.Exit(1)
	}
}
