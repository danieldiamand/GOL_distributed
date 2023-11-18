package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func main() {
	pAddr := flag.String("address", "localhost:8032", "Address to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Broker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Broker struct {
	workers        []*rpc.Client
	workersAdr     []string
	workerSections []int
	workerCount    int

	savedWorld [][]byte
	savedTurn  int
	finalTurn  int

	currentTurn int
	width       int
	height      int
	isPaused    bool
	isQuit      bool
	progressMu  sync.Mutex
}

func (b *Broker) ProgressWorld(req stubs.BrokerProgressWorldReq, res *stubs.WorldRes) (err error) {
	print("Broker created.")
	b.savedWorld = req.World
	b.savedTurn = 0
	b.currentTurn = 0
	b.finalTurn = req.Turns
	b.width = req.Width
	b.height = req.Height

	//Try and connect to each worker
	b.workersAdr = req.WorkersAdr
	b.workerCount = 0
	for _, workerAdr := range b.workersAdr {
		worker, err := rpc.Dial("tcp", workerAdr)
		if err != nil {
			/*
				TODO: handle this error properly, should ask user if they want to
				work with one less user or try reconnecting

				TOOD: rather than having distrubotr having an address have
				different types of response!!!
			*/
			println("Error connecting to worker", err.Error())
		}
		b.workers = append(b.workers, worker)
		b.workerCount++
	}

	//Divide board up into sections to give to each worker,
	//worker i starts at workerSections[i] and finishes at workerSections[i+1]
	sectionLength := b.height / b.workerCount
	remainingLength := b.height % b.workerCount
	b.workerSections = make([]int, b.workerCount+1)
	b.workerSections[0] = 0
	for i := 1; i < b.workerCount+1; i++ {
		b.workerSections[i] = b.workerSections[i-1] + sectionLength
		if i <= remainingLength {
			b.workerSections[i]++
		}
	}

	workerDones := make([]*rpc.Call, b.workerCount)

	//Call Init on each worker
	for i := 0; i < b.workerCount; i++ {
		workerInitReq := stubs.WorkerInitReq{
			World:  b.savedWorld[b.workerSections[i]:b.workerSections[i+1]],
			Width:  b.width,
			Height: b.workerSections[i+1] - b.workerSections[i],
		}
		workerDones[i] = b.workers[i].Go(stubs.WorkerInit, workerInitReq, &stubs.None{}, nil)
	}
	//ensure each init has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker init err", workerDones[i].Error.Error())
		}
		<-workerDones[i].Done
	}

	//Call Start on each worker
	println(b.workersAdr[b.workerCount-1])
	workerStartReq := stubs.WorkerStartReq{AboveAdr: b.workersAdr[b.workerCount-1]} //the top worker connects to the bottom
	workerDones[0] = b.workers[0].Go(stubs.WorkerStart, workerStartReq, &stubs.None{}, nil)
	for i := 1; i < b.workerCount; i++ {
		workerStartReq = stubs.WorkerStartReq{AboveAdr: b.workersAdr[i-1]} //all other workers connect to the one above
		workerDones[i] = b.workers[i].Go(stubs.WorkerStart, workerStartReq, &stubs.None{}, nil)
	}
	//ensure each start has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker start err", workerDones[i].Error.Error())
		}
		<-workerDones[i].Done
	}

	//MAIN LOOP:
	workerTurnRes := make([]stubs.Turn, b.workerCount)
	for b.currentTurn < b.finalTurn && !b.isQuit {
		//Call progressHelper on each worker
		for i := 0; i < b.workerCount; i++ {
			workerDones[i] = b.workers[i].Go(stubs.WorkerProgress, stubs.None{}, &workerTurnRes[i], nil)
		}
		//ensure each start has completed
		for i := 0; i < b.workerCount; i++ {
			if workerDones[i].Error != nil {
				println("Worker progressHelper err", workerDones[i].Error.Error())
				os.Exit(1)
			}
			<-workerDones[i].Done
		}
		b.progressMu.Lock()
		b.currentTurn = workerTurnRes[0].Turn
		b.progressMu.Unlock()
	}

	//Collect final world from workers
	res.Turn, res.World = collectWorldFromWorkers(b)

	//Quit all workers
	for i := 0; i < b.workerCount; i++ {
		workerDones[i] = b.workers[i].Go(stubs.WorkerQuit, stubs.None{}, &stubs.None{}, nil)
	}

	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("quit err", workerDones[i].Error.Error())
		}
		<-workerDones[i].Done
	}

	if b.isQuit {
		println("Broker quit all workers")
	} else {
		println("Broker finished calculating world")
	}

	return
}

func (b *Broker) Count(req stubs.None, res *stubs.CountCellRes) (err error) {
	if b.isPaused {
		res.Count = -1
		return
	}

	workerCountRes := make([]stubs.CountCellRes, b.workerCount)
	workerDones := make([]*rpc.Call, b.workerCount)
	for i := 0; i < b.workerCount; i++ {
		workerDones[i] = b.workers[i].Go(stubs.WorkerCount, stubs.None{}, &workerCountRes[i], nil)
	}

	count := 0
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("count err", workerDones[i].Error.Error())
		}
		<-workerDones[i].Done
		count += workerCountRes[i].Count
		println("worker", i, "on turn", workerCountRes[i].Turn)
	}

	res.Count = count
	res.Turn = workerCountRes[0].Turn
	return
}

func (b *Broker) Pause(req stubs.None, res *stubs.PauseRes) (err error) {
	if !b.isPaused {
		println("Pausing on turn", b.currentTurn)
		b.progressMu.Lock()
		b.isPaused = true
		res.Output = fmt.Sprintf("Pausing on turn %d", b.currentTurn)
	} else {
		println("Unpausing")
		b.isPaused = false
		b.progressMu.Unlock()
		res.Output = "Continuing"
	}
	return
}

func (b *Broker) Fetch(req stubs.None, res *stubs.WorldRes) (err error) {
	res.Turn, res.World = collectWorldFromWorkers(b)
	return
}

func collectWorldFromWorkers(b *Broker) (int, [][]byte) {
	workerDones := make([]*rpc.Call, b.workerCount)
	workerFetchRes := make([]stubs.WorldRes, b.workerCount)
	b.progressMu.Lock()
	for i := 0; i < b.workerCount; i++ {
		workerDones[i] = b.workers[i].Go(stubs.WorkerFetch, stubs.None{}, &workerFetchRes[i], nil)
	}
	b.progressMu.Unlock()

	var world [][]byte
	//ensure each fetch has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker fetch err", workerDones[i].Error.Error())
			os.Exit(1)
		}
		<-workerDones[i].Done
		world = append(world, workerFetchRes[i].World...)
	}
	return workerFetchRes[0].Turn, world
}

func (b *Broker) Quit(req stubs.None, res *stubs.None) (err error) {
	b.isQuit = true
	return
}

func (b *Broker) Kill(req stubs.None, res *stubs.None) (err error) {
	//Quit all workers
	for i := 0; i < b.workerCount; i++ {
		_ = b.workers[i].Go(stubs.WorkerKill, stubs.None{}, &stubs.None{}, nil)
	}
	println("Killed all workers.")
	println("Broker killed.")
	os.Exit(0)
	return
}
