package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gosuri/uilive"

	"conway/engine"
)

const (
	height = 50
	width  = 50
)

func main() {
	universe := engine.NewUniverse(height, width, []engine.UniverseCoord{
		{
			X: 1,
			Y: 2,
		},
		{
			X: 2,
			Y: 3,
		},
		{
			X: 3,
			Y: 1,
		},
		{
			X: 3,
			Y: 2,
		},
		{
			X: 3,
			Y: 3,
		},

		{
			X: 5,
			Y: 6,
		},
		{
			X: 6,
			Y: 6,
		},
		{
			X: 7,
			Y: 6,
		},
	})
	writer := uilive.New()
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
