package api

import "errors"

type Progress struct {
	TotalUnit         uint32
	PendingUnit       uint32
	CompletedUnit     uint32
	CompletedFraction float32
	Child             []*Progress
	Parent            *Progress
	Payload           string
}

func NewProgress(totalUnit uint32) *Progress {
	return &Progress{
		TotalUnit:   totalUnit,
		PendingUnit: totalUnit,
	}
}

func (p *Progress) NewChild(childTotalUnit uint32) (*Progress, error) {
	if childTotalUnit > p.PendingUnit {
		return nil, errors.New("TBD")
	}
	ch := NewProgress(childTotalUnit)
	ch.Parent = p
	p.Child = append(p.Child, ch)
	p.PendingUnit -= childTotalUnit
	return ch, nil
}

func (p *Progress) done(completedUnit uint32) {
	p.PendingUnit -= completedUnit
	p.CompletedUnit += completedUnit
	p.CompletedFraction = float32(p.CompletedUnit) / float32(p.TotalUnit)
}

func (p *Progress) Done(completedUnit uint32) {
	if completedUnit > p.PendingUnit {
		completedUnit = p.PendingUnit
	}
	p.done(completedUnit)

	parent := p.Parent
	for parent != nil {
		parent.done(completedUnit)
		parent = parent.Parent
	}
}
