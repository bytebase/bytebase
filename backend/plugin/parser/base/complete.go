package base

import "fmt"

// CandidateType is the type of candidate.
type CandidateType string

const (
	CandidateTypeNone         CandidateType = "NONE"
	CandidateTypeKeyword      CandidateType = "KEYWORD"
	CandidateTypeDatabase     CandidateType = "DATABASE"
	CandidateTypeSchema       CandidateType = "SCHEMA"
	CandidateTypeTable        CandidateType = "TABLE"
	CandidateTypeRoutine      CandidateType = "ROUTINE"
	CandidateTypeFunction     CandidateType = "FUNCTION"
	CandidateTypeView         CandidateType = "VIEW"
	CandidateTypeColumn       CandidateType = "COLUMN"
	CandidateTypeOperator     CandidateType = "OPERATOR"
	CandidateTypeEngine       CandidateType = "ENGINE"
	CandidateTypeTrigger      CandidateType = "TRIGGER"
	CandidateTypeLogFileGroup CandidateType = "LOGFILEGROUP"
	CandidateTypeUserVar      CandidateType = "USERVAR"
	CandidateTypeSystemVar    CandidateType = "SYSTEMVAR"
	CandidateTypeTableSpace   CandidateType = "TABLESPACE"
	CandidateTypeEvent        CandidateType = "EVENT"
	CandidateTypeIndex        CandidateType = "INDEX"
	CandidateTypeUser         CandidateType = "USER"
	CandidateTypeCharset      CandidateType = "CHARSET"
	CandidateTypeCollation    CandidateType = "COLLATION"
)

// Candidate is the candidate for auto-completion.
type Candidate struct {
	Text       string
	Type       CandidateType
	Definition string
	Comment    string
}

func (c Candidate) String() string {
	return fmt.Sprintf("%s (%s)", c.Text, c.Type)
}

type TableReference interface {
	isTableReference()
}

type PhysicalTableReference struct {
	Database string
	Table    string
	Alias    string
}

func (*PhysicalTableReference) isTableReference() {}

type VirtualTableReference struct {
	Table   string
	Columns []string
}

func (*VirtualTableReference) isTableReference() {}
