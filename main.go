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
		"brokerAddress",
		"localhost:8032",
		"The address of Broker. Defaults to localhost:8032")
	workerAddresses := flag.String(
		"workerAddresses",
		"localhost:8030,localhost:8031",
		"The addresses of Workers seperated by a comma, will throw if less than threads. Defaults to localhost:8030")

	noVis := flag.Bool(
		"noVis",
		false,
		"Disables the SDL window, so there is no visualisation during the tests.")

	printProgress := flag.Bool(
		"printProgress",
		false,
		"Workers and Broker print out each turn of world for debugging purposes (only works for low turns and short worlds)")

	flag.Parse()

	params.PrintProgress = *printProgress
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
