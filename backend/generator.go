/*
Random or seeded generator for boxes arriving into the queue at time stamps
*/

package backend

import (
	"errors"
	"math/rand"
)

type BoxGenerator interface {
	Next(arrivedAt int) QueuedBox
}

type RandomBoxGenerator struct {
	rng                  *rand.Rand
	minWidth, maxWidth   int
	minHeight, maxHeight int
	nextID               int
}

func NewRandomBoxGenerator(
	seed int64,
	minWidth, maxWidth int,
	minHeight, maxHeight int,
) (*RandomBoxGenerator, error) {
	if minWidth <= 0 || minHeight <= 0 {
		return nil, errors.New("minimum box dimensions must be positive")
	}
	if maxWidth < minWidth || maxHeight < minHeight {
		return nil, errors.New("maximum box dimensions must be at least the minimum dimensions")
	}

	return &RandomBoxGenerator{
		rng:       rand.New(rand.NewSource(seed)),
		minWidth:  minWidth,
		maxWidth:  maxWidth,
		minHeight: minHeight,
		maxHeight: maxHeight,
		nextID:    1,
	}, nil
}

func (g *RandomBoxGenerator) Next(t int) QueuedBox {
	width := g.minWidth + g.rng.Intn(g.maxWidth-g.minWidth+1)
	height := g.minHeight + g.rng.Intn(g.maxHeight-g.minHeight+1)

	box := Box{
		ID:     g.nextID,
		Width:  width,
		Height: height,
	}
	g.nextID++

	return QueuedBox{Box: box, ArrivedAt: t}
}
