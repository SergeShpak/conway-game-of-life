package engine

import (
	"math"
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
	Field      map[UniverseCoord]struct{}
	nextField  map[UniverseCoord]struct{}
	virtifield *virtfield
	rules      *UniverseRules
	readMux    sync.RWMutex
}

type UniverseCoord struct {
	X int32
	Y int32
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

func NewUniverse(height uint32, width uint32, filledCells []UniverseCoord) *Universe {
	field := make(map[UniverseCoord]struct{})
	u := &Universe{
		Field:      field,
		rules:      getDefaultUniverseRules(),
		virtifield: newVirtfield(height, width),
	}
	for _, cell := range filledCells {
		normalizedCell := u.virtifield.NormalizeUniverseCoord(cell)
		u.set(normalizedCell)
	}
	return u
}

func (u *Universe) set(coord UniverseCoord) {
	u.Field[coord] = struct{}{}
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
	if aliveNeighbors == u.rules.Reproduction {
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
	var i, j int32
	for i = -1; i <= 1; i++ {
		for j = -1; j <= 1; j++ {
			if i == 0 && j == 0 {
				continue
			}
			if isOverflowed(coord.X, i) || isOverflowed(coord.Y, j) {
				continue
			}
			neighbor := UniverseCoord{
				X: coord.X + i,
				Y: coord.Y + j,
			}
			normalizedNeighbor := u.virtifield.NormalizeUniverseCoord(neighbor)
			neighborsCoords = append(neighborsCoords, normalizedNeighbor)
		}
	}
	return neighborsCoords
}

func isOverflowed(coord int32, diff int32) bool {
	if diff == 0 {
		return false
	}
	if diff < 0 {
		return coord < (math.MinInt32 - diff)
	}
	// diff > 0
	return coord > (math.MaxInt32 - diff)
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
