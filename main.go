package main

import (
	"flag"
	"fmt"
	"runtime"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/sdl"
)

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()
	var params gol.Params

	flag.IntVar(
		&params.Threads,
		"t",
		8,
		"Specify the number of worker threads to use. Defaults to 8.")

	flag.IntVar(
		&params.ImageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.ImageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")

	flag.IntVar(
		&params.Turns,
		"turns",
		10000000000,
		"Specify the number of turns to process. Defaults to 10000000000.")

	brokerAddress := flag.String(
		"brokerAdr",
		"184.73.149.30:8030",
		"The address of Broker. Defaults to localhost:8032")
	workerAddresses := flag.String(
		"workerAddresses",
		"18.207.208.4:8030,18.233.169.12:8030,3.83.228.73:8030,34.205.157.114:8030",
		"The addresses of Workers seperated by a space. Defaults to localhost:8030")

	noVis := flag.Bool(
		"noVis",
		false,
		"Disables the SDL window, so there is no visualisation during the tests.")

	flag.Parse()

	params.BrokerAddress = *brokerAddress
	params.WorkerAddresses = *workerAddresses
	fmt.Println("Threads:", params.Threads)
	fmt.Println("Width:", params.ImageWidth)
	fmt.Println("Height:", params.ImageHeight)

	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)

	go gol.Run(params, events, keyPresses)
	if !(*noVis) {
		sdl.Run(params, events, keyPresses)
	} else {
		complete := false
		for !complete {
			event := <-events
			switch event.(type) {
			case gol.FinalTurnComplete:
				complete = true
			}
		}
	}
}
