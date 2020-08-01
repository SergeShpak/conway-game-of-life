package engine

import (
	"math"
)

type virtfield struct {
	height int32
	width  int32
}

func newVirtfield(height uint32, width uint32) *virtfield {
	var normHeight, normWidth int32
	if height < uint32(math.MaxInt32) {
		normHeight = int32(height)
	} else {
		normHeight = math.MaxInt32
	}
	if width < uint32(math.MaxInt32) {
		normWidth = int32(width)
	} else {
		normWidth = math.MaxInt32
	}
	vf := &virtfield{
		height: normHeight,
		width:  normWidth,
	}
	return vf
}

func (vf *virtfield) NormalizeField(f map[UniverseCoord]struct{}) map[UniverseCoord]struct{} {
	normalizedField := make(map[UniverseCoord]struct{}, len(f))
	for coord := range f {
		normalizedCoord := vf.NormalizeUniverseCoord(coord)
		normalizedField[normalizedCoord] = struct{}{}
	}
	return normalizedField
}

func (vf *virtfield) NormalizeUniverseCoord(c UniverseCoord) UniverseCoord {
	normCoord := UniverseCoord{}
	normCoord.X = normalizeDim(c.X, vf.height)
	normCoord.Y = normalizeDim(c.Y, vf.width)
	return normCoord
}

func normalizeDim(coord int32, len int32) int32 {
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
