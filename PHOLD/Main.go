/*
	Go-based implementation of the PHOLD synthetic benchmark
	http://pads.cs.unibo.it

	This file is part of GO-WARP.  GO-WARP is free software, you can
	redistribute it and/or modify it under the terms of the Revised BSD License.

	For more information please see the LICENSE file.

	Copyright 2014, Gabriele D'Angelo, Moreno Marzolla, Pietro Ansaloni
	Computer Science Department, University of Bologna, Italy
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/jeffallen/go-warp/lcg16807"
	"github.com/jeffallen/go-warp/warp"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	usage  = "Main.out #LPs [if 0 -> autoconf] #ENTITIES"
	conf   = "./phold.conf"
	cpustr = "processor"
)

var (
	lpnum     int
	entitynum int
	density   float64
	n_events  int
	endtime   warp.Time
	nFPops    int
	randGen   *lcg16807.RNG

	initEv []warp.Event

	idcount int32

	startT   time.Time
	elapsedT time.Duration
	simterm  bool
	n_term   int
	print    sync.Mutex

	n_cores int
)

func main() {
	n_lp, n_ent := readParams()

	initPhold(n_lp, n_ent)

	fmt.Println("GO-WARP: the simulator will use", runtime.GOMAXPROCS(-1), "COREs")
	fmt.Println("GO-WARP: the simulation will use", n_lp, "LPs")

	startT = time.Now()
	for i := 1; i < n_lp; i++ {
		go launchLP(warp.Pid(i), n_ent/n_lp)
	}
	launchLP(0, n_ent/n_lp)

	for n_term != lpnum {
		time.Sleep(1e3)
	}
	printStats(elapsedT)
}

func readParams() (nlp int, nent int) {
	flag.Parse()
	narg := flag.NArg()
	if narg != 2 {
		fmt.Printf("%s\n", usage)
		os.Exit(1)
	}
	nlp, _ = strconv.Atoi(flag.Arg(0))
	nent, _ = strconv.Atoi(flag.Arg(1))

	if nlp == 0 {
		nlp = runtime.NumCPU()
		fmt.Println("GO-WARP: value 0 as LP number, it means that the simulator will use all the available CPU cores")
	}

	fmt.Println("PARAMS:", nlp, nent)
	return nlp, nent
}

func initPhold(nlp int, nent int) {
	var rdErr error
	var line string
	var str []string = make([]string, 2)

	file, err := os.Open(conf)
	if err != nil {
		fmt.Println("GO-WARP, error opening the PHOLD configuration file:", err)
		os.Exit(1)
	}
	rd := bufio.NewReader(file)

	stop := false
	for i := 0; rdErr == nil; i++ {
		line, rdErr = rd.ReadString('\n')
		if len(line) > 0 {
			str = strings.SplitN(line, "#", 2)
			num := strings.Replace(str[0], "\t", "", -1)
			num = strings.Replace(num, " ", "", -1)
			if i == 0 {
				var err error
				density, err = strconv.ParseFloat(num, 64)
				if err != nil {
					fmt.Printf("line %v: error parsing float %v: %v", i, num, err)
					stop = true
					break
				}
			} else if i == 1 {
				t, err := strconv.Atoi(num)
				if err != nil {
					fmt.Printf("line %v: error parsing int %v: %v", i, num, err)
					stop = true
					break
				}
				endtime = warp.Time(t)
			} else if i == 2 {
				nFPops, err = strconv.Atoi(num)
				if err != nil {
					fmt.Printf("line %v: error parsing int %v: %v", i, num, err)
					stop = true
					break
				}
			}
		}
	}
	if stop {
		fmt.Println("GO-WARP: config file parse error.")
		os.Exit(1)
	}
	fmt.Println("GO-WARP: read from file:", density, endtime, nFPops)

	lpnum = nlp
	entitynum = nent
	n_events = int(float64(nent) * density)
	randGen = lcg16807.RandInit(int64(lpnum + entitynum))
	idcount = 0

	initEv = make([]warp.Event, n_events)

	warp.SimSetup(lpnum, endtime, ProcessEvent)

	for i := 0; i < n_events; i++ {
		e := generateEvent(nil)
		initEv[i] = *e
	}

	simterm = false
	n_term = 0
}

func launchLP(index warp.Pid, n_entity int) {
	var data *warp.LocalData
	data = warp.SimInitialize(index)

	getEvents(index, data)

	warp.Simulate(data)

	terminate(data)
}

// each event in the system is generated in this function
func generateEvent(oldev *warp.Event) *warp.Event {
	var mitt int
	var dest int
	var id int32
	var t warp.Time

	if oldev == nil {
		mitt = int(randGen.RandIntUniform(0, int32(entitynum)))
		t = 0 // basetime
	} else {
		mitt = oldev.Type.To
		t = oldev.Time // basetime
	}

	dest = int(randGen.RandIntUniform(0, int32(entitynum-1)))
	for mitt == dest {
		dest = int(randGen.RandIntUniform(0, int32(entitynum-1)))
	}
	id = idcount
	idcount++
	t += warp.Time(randGen.RandIntExponential())

	e := warp.CreateEvent(id, t, warp.Info{mitt, dest, 0})
	return e
}

// each LP gets his events from those that have been generated at start up
func getEvents(index warp.Pid, data *warp.LocalData) {
	for i := 0; i < n_events; i++ {
		if warp.Pid(initEv[i].Type.From/(entitynum/lpnum)) == index {
			data.FutureEvents.Insert(&initEv[i])
		}
	}
}

func ProcessEvent(ev *warp.Event, l *warp.LocalData) {
	newev := generateEvent(ev)
	lp := e2lp(newev.Type.To, entitynum, lpnum)
	warp.NoticeEvent(newev, warp.Pid(lp), l)
	compute()
}

func e2lp(e, en, lpn int) int {
	var lp int
	m := en % lpn
	d := en / lpn
	if m == 0 {
		lp = e / d
	} else {
		lp = e / (d + 1)
		if lp >= m {
			ee := e - m*(d+1)
			lp = m + ee/d
		}
	}
	return lp
}

func compute() float64 {
	var z, x float64
	z = 2
	x = 0.5

	for i := 0; i < nFPops/5; i++ {
		x = 0.5 * x * (3 - z*x*x)
	}
	return x
}

func terminate(data *warp.LocalData) {
	if !simterm {
		simterm = true
		elapsedT = time.Now().Sub(startT)
	}

	print.Lock()

	fmt.Println("|----------------------------------------------|")
	fmt.Println("LOGICAL PROCESS", data.IndexLP)
	fmt.Println("Number of processed events =", data.N_PROCESSED)

	n_term++
	print.Unlock()
}

func printStats(elapsed time.Duration) {
	print.Lock()

	fmt.Println("SIMULATION IS COMPLETED: TIME REACHED VALUE", endtime)
	fmt.Println("Wall Clock Time spent (ms):", elapsed/time.Millisecond)

	fmt.Println("Number of GVT evaluations:", warp.N_gvt)

	sum := 0
	for i := 0; i < lpnum; i++ {
		sum += warp.N_rollback[i]
	}
	fmt.Println("Total number of rollbacks:", sum)

	print.Unlock()
}
