package main

import (
	"errors"
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
	if err != nil {
		println("Error in Broker registering: ", err.Error())
		return
	}
	listener, _ := net.Listen("tcp", *pAddr)
	if err != nil {
		println("Error in Broker listening: ", err.Error())
		return
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			println("Error closing Broker")
			return
		}
	}(listener)
	rpc.Accept(listener)
}

type Broker struct {
	world       [][]byte
	currentTurn int
	finalTurn   int
	width       int
	height      int

	printProgress bool

	workers        []*rpc.Client
	workersAdr     []string
	workerSections []int
	workerCount    int

	isPaused   bool
	isQuit     bool
	progressMu sync.Mutex
}

func (b *Broker) Init(req stubs.BrokerInitReq, res *stubs.None) (err error) {

	b.world = req.World
	b.currentTurn = 0
	b.finalTurn = req.Turns
	b.width = req.Width
	b.height = req.Height
	b.printProgress = req.PrintProgress

	println("Broker created, on world", b.width, "x", b.height, ".")
	if b.printProgress {
		println("World at init. Turn:", b.currentTurn)
		util.VisualiseMatrix(b.world, b.width, b.height)
	}

	return
}

func (b *Broker) Start(req stubs.BrokerStartReq, res *stubs.None) (err error) {
	//rpc Dial each worker
	b.workers = make([]*rpc.Client, 0)
	b.workersAdr = make([]string, 0)
	b.workerCount = 0
	for _, workerAdr := range req.WorkerAddresses {
		worker, err := rpc.Dial("tcp", workerAdr)
		if err != nil {
			return errors.New(fmt.Sprint("Error in Broker connecting to Worker: ", err.Error()))
		}
		b.workers = append(b.workers, worker)
		b.workersAdr = append(b.workersAdr, workerAdr)
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

	//Distribute world to workers
	workerDones := make([]*rpc.Call, b.workerCount)

	//Call Init on each worker
	for i := 0; i < b.workerCount; i++ {
		workerInitReq := stubs.WorkerInitReq{
			World:         b.world[b.workerSections[i]:b.workerSections[i+1]],
			Width:         b.width,
			Height:        b.workerSections[i+1] - b.workerSections[i],
			PrintProgress: b.printProgress,
		}
		workerDones[i] = b.workers[i].Go(stubs.WorkerInit, workerInitReq, &stubs.None{}, nil)
	}
	//ensure each Init has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker init err", workerDones[i].Error.Error())
			return errors.New(fmt.Sprint("Error in Broker calling Init on Worker: ", workerDones[i].Error.Error()))
		}
		<-workerDones[i].Done
	}

	//Call Start on each worker
	workerStartReq := stubs.WorkerStartReq{AboveAdr: b.workersAdr[b.workerCount-1]} //the top worker connects to the bottom
	workerDones[0] = b.workers[0].Go(stubs.WorkerStart, workerStartReq, &stubs.None{}, nil)
	for i := 1; i < b.workerCount; i++ {
		workerStartReq = stubs.WorkerStartReq{AboveAdr: b.workersAdr[i-1]} //all other workers connect to the one above
		workerDones[i] = b.workers[i].Go(stubs.WorkerStart, workerStartReq, &stubs.None{}, nil)
	}
	//ensure each Start has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker start err", workerDones[i].Error.Error())
			return errors.New(fmt.Sprint("Error in Broker calling Start on Worker: ", workerDones[i].Error.Error()))
		}
		<-workerDones[i].Done
	}

	return //Return no error
}

func (b *Broker) ProgressAll(req stubs.WorldRes, res *stubs.None) (err error) {

	//MAIN LOOP:
	workerTurnRes := make([]stubs.Turn, b.workerCount)
	workerDones := make([]*rpc.Call, b.workerCount)
	for b.currentTurn < b.finalTurn && !b.isQuit {
		//Call progressHelper on each worker
		for i := 0; i < b.workerCount; i++ {
			workerDones[i] = b.workers[i].Go(stubs.WorkerProgress, stubs.None{}, &workerTurnRes[i], nil)
		}
		//ensure each start has completed
		for i := 0; i < b.workerCount; i++ {
			if workerDones[i].Error != nil {
				return errors.New(fmt.Sprint("Error in Broker calling Progress on Worker: ", workerDones[i].Error.Error()))
			}
			<-workerDones[i].Done
		}
		b.progressMu.Lock()
		b.currentTurn = workerTurnRes[0].Turn
		b.progressMu.Unlock()
	}

	println("Broker finished calculating world at turn", b.currentTurn, "out of", b.finalTurn)

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
			return errors.New(fmt.Sprint("Error in Broker calling Count on Worker: ", workerDones[i].Error.Error()))
		}
		<-workerDones[i].Done
		count += workerCountRes[i].Count
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
	res.Turn, res.World, err = collectWorldFromWorkers(b)
	if err != nil {
		return errors.New(fmt.Sprint("Error in Broker calling Fetch on Worker: ", err.Error()))
	}
	if b.printProgress {
		println("World at fetch. Turn:", b.currentTurn)
		util.VisualiseMatrix(res.World, b.width, b.height)
	}
	return
}

func collectWorldFromWorkers(b *Broker) (int, [][]byte, error) {
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
			return 0, nil, errors.New(fmt.Sprint(workerDones[i].Error.Error()))
		}
		<-workerDones[i].Done
		world = append(world, workerFetchRes[i].World...)
	}
	return workerFetchRes[0].Turn, world, nil
}

func (b *Broker) Quit(req stubs.None, res *stubs.None) (err error) {
	//Quit all workers
	println("Broker quit.")
	b.isQuit = true //stops progressAll loop
	return
}

func (b *Broker) Kill(req stubs.None, res *stubs.None) (err error) {
	//Kill all workers
	for i := 0; i < b.workerCount; i++ {
		_ = b.workers[i].Go(stubs.WorkerKill, stubs.None{}, &stubs.None{}, nil)
	}
	println("Killed all workers.")
	println("Broker killed.")
	os.Exit(0)
	return
}
