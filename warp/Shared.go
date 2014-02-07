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

import (
	"fmt"
	"time"
)

var (
	Lpnum        int
	N_gvt        int
	State        []int8
	N_rollback   []int
	EventManager func(ev *Event, l *LocalData)
	EndTime      Time

	StartTime time.Time
)

func SharedSetup(lpn int, simt Time, f func(ev *Event, l *LocalData)) {
	Lpnum = lpn
	EndTime = simt
	N_gvt = 0
	State = make([]int8, lpn)
	N_rollback = make([]int, lpn)
	for i := 0; i < lpn; i++ {
		State[i] = LPNOTSTART
		N_rollback[i] = 0
	}
	EventManager = f

	fmt.Println("SETUP COMPLETED: lpn =", Lpnum, "EndTime =", EndTime)
	StartTime = time.Now()
}
