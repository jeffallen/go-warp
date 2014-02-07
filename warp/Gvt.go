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
	"sync"
)

var (
	lpNum    int
	localMin []Time
	gvt      Time
	gvtFlag  bool
	gvtlock  sync.Mutex
)

const MAXTIME = 1<<31 - 1
const EMPTY = -13

func GvtSetup(lpnum int) {

	localMin = make([]Time, lpnum)
	lpNum = lpnum
	gvt = 0
	gvtFlag = false
}

func StartEvaluation(lpnum int) {
	if !gvtFlag {
		for i := 0; i < lpnum; i++ {
			localMin[i] = EMPTY
		}
		lpNum = lpnum
		gvtFlag = true
	}
}

/* if true the a GVT calculation is running */
func CheckEvaluation() bool {
	return gvtFlag
}

func SetLocalMin(time Time, pid Pid) {

	if !gvtFlag {
		return
	}

	localMin[pid] = time

	for i := 0; i < len(localMin); i++ {
		if localMin[i] == EMPTY {
			return
		}
	}

	gvtlock.Lock()
	setGVT()
	gvtlock.Unlock()
}

func setGVT() {

	for i := 0; i < len(localMin); i++ {
		if localMin[i] == EMPTY {
			return
		}
	}

	tmpMin := Time(MAXTIME)
	for i := 0; i < len(localMin); i++ {
		if localMin[i] < tmpMin && localMin[i] != NOTIME {
			tmpMin = localMin[i]
		}
	}
	gvt = tmpMin

	for i := 0; i < len(localMin); i++ {
		localMin[i] = EMPTY
	}
	gvtFlag = false
	N_gvt++
}

func GetGvt() Time {

	if gvtFlag {
		return ERR
	}
	return gvt
}
