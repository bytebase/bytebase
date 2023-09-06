package v1

import v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

type metadataDiffAction string

const (
	metadataDiffActionCreate metadataDiffAction = "CREATE"
	metadataDiffActionUpdate metadataDiffAction = "UPDATE"
	metadataDiffActionDrop   metadataDiffAction = "DROP"
)

type metadataSchemaDiff struct {
	action metadataDiffAction
	from   *v1pb.SchemaMetadata
	to     *v1pb.SchemaMetadata

	tableDiffs map[string]*metadataTableDiff
}

type metadataTableDiff struct {
	action metadataDiffAction
	from   *v1pb.TableMetadata
	to     *v1pb.TableMetadata

	columnDiffs map[string]*metadataColumnDiff
}

type metadataColumnDiff struct {
	action metadataDiffAction
	from   *v1pb.ColumnMetadata
	to     *v1pb.ColumnMetadata
}

type metadataDiff struct {
	schemaDiffs map[string]*metadataSchemaDiff
}

// computeDiffBetweenMetadata computes the difference between two database metadata.
func computeDiffBetweenMetadata(from, to *v1pb.DatabaseMetadata) (*metadataDiff, error) {

}
