package mysql

import "fmt"

// PartitionDefaultNameGenerator is the name generator of MySQL partition, which use the default clause.
// The behavior of this generator should be compatible with MySQL.
// - If do not specify the `parentName`, the default partition name series is "p0", "p1", "p2", ...
// - Otherwise, the default partition name series is "parentNamesp0", "parentNamesp1", "parentNamesp2", ...
type PartitionDefaultNameGenerator struct {
	parentName string
	count      int
}

func NewPartitionDefaultNameGenerator(parentName string) *PartitionDefaultNameGenerator {
	return &PartitionDefaultNameGenerator{
		parentName: parentName,
		count:      -1,
	}
}

func (g *PartitionDefaultNameGenerator) Next() string {
	g.count++

	if g.parentName == "" {
		return fmt.Sprintf("p%d", g.count)
	}
	return fmt.Sprintf("%ssp%d", g.parentName, g.count)
}
