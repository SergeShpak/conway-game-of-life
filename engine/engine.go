package engine

import (
	"fmt"
	"sync"
)

const (
	ruleOverpopulation = 4
	ruleStarvation     = 1

	ruleReproduction = 3
)

type UniverseRules struct {
	Overpopulation int
	Starvation     int
	Reproduction   int
}

func getDefaultUniverseRules() *UniverseRules {
	return &UniverseRules{
		Overpopulation: ruleOverpopulation,
		Starvation:     ruleStarvation,
		Reproduction:   ruleReproduction,
	}
}

type Universe struct {
	Field     map[UniverseCoord]struct{}
	nextField map[UniverseCoord]struct{}
	Height    int
	Width     int
	rules     *UniverseRules
	readMux   sync.RWMutex
}

type UniverseCoord struct {
	X int
	Y int
}

type UniverseCell struct {
	Coord  UniverseCoord
	Filled bool
}

type DeadCellAssignment struct {
	cells map[UniverseCoord]struct{}
	mux   sync.Mutex
}

func newDeadCellAssignement() *DeadCellAssignment {
	a := &DeadCellAssignment{
		cells: make(map[UniverseCoord]struct{}),
	}
	return a
}

type Neighbors struct {
	Neighbors map[UniverseCoord]UniverseCell
}

func (a *DeadCellAssignment) Assign(coord UniverseCoord) bool {
	a.mux.Lock()
	if _, ok := a.cells[coord]; ok {
		a.mux.Unlock()
		return false
	}
	a.cells[coord] = struct{}{}
	a.mux.Unlock()
	return true
}

func NewUniverse(height int, width int, filledCells []UniverseCoord) (*Universe, error) {
	if height <= 0 || width <= 0 {
		return nil, fmt.Errorf("universe height and width must be positive integers")
	}
	field := make(map[UniverseCoord]struct{})
	u := &Universe{
		Field:  field,
		Height: height,
		Width:  width,
		rules:  getDefaultUniverseRules(),
	}
	for _, cell := range filledCells {
		u.set(cell)
	}
	return u, nil
}

func (u *Universe) set(coord UniverseCoord) {
	normalizedCoord := u.normalizeCoord(coord)
	u.Field[normalizedCoord] = struct{}{}
}

func (u *Universe) unset(coord UniverseCoord) {
	normalizedCoord := u.normalizeCoord(coord)
	delete(u.Field, normalizedCoord)
}

func (u *Universe) Step() {
	assignment := newDeadCellAssignement()
	const diffChanBuf = 100
	diffChan := make(chan UniverseCell, 100)
	var collectorWG sync.WaitGroup
	collectorWG.Add(1)
	go func() {
		u.composeNewField(diffChan)
		collectorWG.Done()
	}()
	var workersWG sync.WaitGroup
	for coord := range u.Field {
		workersWG.Add(1)
		coord := coord
		go u.evaluateLivingCell(coord, diffChan, assignment, &workersWG)
	}
	workersWG.Wait()
	close(diffChan)
	collectorWG.Wait()
	u.Field = u.nextField
	u.nextField = nil
}

func (u *Universe) composeNewField(diffChan <-chan UniverseCell) {
	u.nextField = make(map[UniverseCoord]struct{})
	var isFinished bool
	for {
		if isFinished {
			break
		}
		select {
		case cell, more := <-diffChan:
			if cell.Filled {
				u.nextField[cell.Coord] = struct{}{}
			}
			isFinished = !more
		}
	}
	return
}

func (u *Universe) evaluateLivingCell(coord UniverseCoord, diffChan chan<- UniverseCell, assignment *DeadCellAssignment, wg *sync.WaitGroup) {
	neighbors := u.getNeighbors(coord)
	var aliveCount int
	for _, n := range neighbors.Neighbors {
		if n.Filled {
			aliveCount++
		}
		if !n.Filled {

			wg.Add(1)
			go u.evaluateDeadCell(n.Coord, neighbors, diffChan, assignment, wg)
		}
	}
	if aliveCount > u.rules.Starvation && aliveCount < u.rules.Overpopulation {
		diffChan <- UniverseCell{
			Coord:  coord,
			Filled: true,
		}
	}
	wg.Done()
}

func (u *Universe) evaluateDeadCell(coord UniverseCoord, neighbors *Neighbors, diffChan chan<- UniverseCell, assignment *DeadCellAssignment, wg *sync.WaitGroup) {
	if isAssigned := assignment.Assign(coord); !isAssigned {
		wg.Done()
		return
	}
	neighborsCoord := u.getNeighborsCoords(coord)
	var aliveNeighbors int
	for _, c := range neighborsCoord {
		if cachedNeighbor, ok := neighbors.Neighbors[c]; ok {
			if cachedNeighbor.Filled {
				aliveNeighbors++
			}
			continue
		}
		neighbor := u.getCell(c)
		if neighbor.Filled {
			aliveNeighbors++
		}
	}
	if aliveNeighbors >= u.rules.Reproduction {
		diffChan <- UniverseCell{
			Coord:  coord,
			Filled: true,
		}
	}
	wg.Done()
}

func (u *Universe) getNeighbors(coord UniverseCoord) *Neighbors {
	neighborsCoords := u.getNeighborsCoords(coord)
	neighbors := &Neighbors{
		Neighbors: make(map[UniverseCoord]UniverseCell, 8),
	}
	for _, c := range neighborsCoords {
		n := u.getCell(c)
		neighbors.Neighbors[n.Coord] = n
	}
	return neighbors
}

func (u *Universe) getNeighborsCoords(coord UniverseCoord) []UniverseCoord {
	neighborsCoords := make([]UniverseCoord, 0, 8)
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i == 0 && j == 0 {
				continue
			}
			neighbor := UniverseCoord{
				X: coord.X + i,
				Y: coord.Y + j,
			}
			normalizedNeighbor := u.normalizeCoord(neighbor)
			neighborsCoords = append(neighborsCoords, normalizedNeighbor)
		}
	}
	return neighborsCoords
}

func (u *Universe) normalizeCoord(coord UniverseCoord) UniverseCoord {
	x := getCoordWithWrap(coord.X, u.Height)
	y := getCoordWithWrap(coord.Y, u.Width)
	normalizedCoord := UniverseCoord{
		X: x,
		Y: y,
	}
	return normalizedCoord
}

func (u *Universe) getCell(coord UniverseCoord) UniverseCell {
	cell := UniverseCell{
		Coord: coord,
	}
	u.readMux.RLock()
	if _, ok := u.Field[coord]; ok {
		cell.Filled = true
	}
	u.readMux.RUnlock()
	return cell
}

func getCoordWithWrap(coord int, len int) int {
	if coord >= 0 && coord < len {
		return coord
	}
	if len <= 1 {
		return 0
	}
	if coord < 0 {
		dist := (len - 1) - coord
		wrappedCoord := (dist/len)*len + coord
		return wrappedCoord
	}
	// coord >= len
	return coord % len
}
