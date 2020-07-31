package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gosuri/uilive"

	"conway/engine"
)

const (
	height = 5
	width  = 5
)

func main() {
	universe, err := engine.NewUniverse(height, width, []engine.UniverseCoord{
		{
			X: 1,
			Y: 2,
		},
		{
			X: 2,
			Y: 2,
		},
		{
			X: 3,
			Y: 2,
		},
	})
	if err != nil {
		log.Fatalf("failed to create a universe: %v", err)
	}
	writer := uilive.New()
	// start listening for updates and render
	writer.Start()
	for {
		repr := buildRepr(universe.Field)
		fmt.Fprintln(writer, repr)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			universe.Step()
			wg.Done()
		}()
		time.Sleep(time.Millisecond * 500)
		wg.Wait()
	}
}

func buildRepr(field map[engine.UniverseCoord]struct{}) string {
	repr := make([][]byte, height)
	for i := range repr {
		repr[i] = make([]byte, width)
		for j := 0; j < width; j++ {
			repr[i][j] = '-'
		}
	}
	for coord := range field {
		repr[coord.X][coord.Y] = 'X'
	}
	results := make([]string, height)
	for i, l := range repr {
		results[i] = string(l)
	}
	return strings.Join(results, "\n")
}
