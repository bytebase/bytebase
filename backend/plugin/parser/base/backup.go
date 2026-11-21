package base

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type BackupStatement struct {
	Statement       string
	SourceSchema    string
	SourceTableName string
	TargetTableName string

	StartPosition *storepb.Position
	EndPosition   *storepb.Position
}

// ConvertBackupStatements maps engine backup items onto Bytebase BackupStatements
// using positions from splitSQL aligned with each segment's byte offsets.
func ConvertBackupStatements[S, B any](
	statement string,
	splitSQL func(string) ([]Statement, error),
	segs []S,
	segBounds func(S) (start, end int),
	backups []B,
	extract func(B) (stmt, srcTable, bakTable string, start, end int),
) ([]BackupStatement, error) {
	list, err := splitSQL(statement)
	if err != nil {
		return nil, err
	}
	starts := make(map[int]*storepb.Position, len(segs))
	ends := make(map[int]*storepb.Position, len(segs))
	for i, seg := range segs {
		s, e := segBounds(seg)
		starts[s] = list[i].Start
		ends[e] = list[i].End
	}
	result := make([]BackupStatement, 0, len(backups))
	for _, b := range backups {
		stmt, src, bak, s, e := extract(b)
		result = append(result, BackupStatement{
			Statement:       stmt,
			SourceTableName: src,
			TargetTableName: bak,
			StartPosition:   starts[s],
			EndPosition:     ends[e],
		})
	}
	return result, nil
}
