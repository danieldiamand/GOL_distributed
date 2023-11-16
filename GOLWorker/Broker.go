package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Broker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", "localhost:"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Broker struct {
	world    [][]byte
	workers  []*rpc.Client
	turn     int
	width    int
	height   int
	isPaused bool
	isQuit   bool
	worldMu  sync.Mutex
}

func (w *Broker) ProgressWorld(progressReq stubs.BrokerProgressWorldReq, progressRes *stubs.WorldRes) (err error) {
	//Try and connect to all workers
	for _, workerAdr := range progressReq.WorkersAdr {
		worker, err := rpc.Dial("tcp", workerAdr)
		if err != nil {
			println("Error connecting to worker", err.Error())
		}
		w.workers = append(w.workers, worker)
	}

	w.world = progressReq.World
	w.width = progressReq.W
	w.height = progressReq.H
	w.turn = 0
	w.isQuit = false
	w.isPaused = false

	println("Broker progressing world: ", w.width, "x", w.height)
	for w.turn < progressReq.Turns && !w.isQuit {
		worldRequest := stubs.WorkerProgressWorldReq{World: w.world, W: w.width, H: w.height}
		worldResponse := new(stubs.WorldRes)
		err := w.workers[0].Call(stubs.WorkerProgressWorld, worldRequest, &worldResponse)

		if err != nil {
			println("worker progress world err", err.Error())
		}

		w.worldMu.Lock()
		w.world = worldResponse.World
		w.turn++
		w.worldMu.Unlock()
	}

	progressRes.World = w.world
	progressRes.Turn = w.turn
	println("Broker finished progressing world")

	return
}
