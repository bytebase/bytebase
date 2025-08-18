package ast

// IndexMethodType is the index method type.
type IndexMethodType int

const (
	// IndexMethodTypeBTree is the index method type for B-Tree. It's the default type.
	IndexMethodTypeBTree = iota
	// IndexMethodTypeHash is the index method type for hash.
	IndexMethodTypeHash
	// IndexMethodTypeGiST is the index method type for GiST.
	IndexMethodTypeGiST
	// IndexMethodTypeSpGiST is the index method type for SP-GiST.
	IndexMethodTypeSpGiST
	// IndexMethodTypeGin is the index method type for GIN.
	IndexMethodTypeGin
	// IndexMethodTypeBrin is the index method type for BRIN.
	IndexMethodTypeBrin
	// IndexMethodTypeIvfflat is the index method type for ivfflat.
	// https://github.com/bytebase/bytebase/issues/6783.
	// https://github.com/pgvector/pgvector.
	IndexMethodTypeIvfflat
)

// String implements fmt.Stringer interface.
func (tp IndexMethodType) String() string {
	switch tp {
	case IndexMethodTypeBTree:
		return "btree"
	case IndexMethodTypeHash:
		return "hash"
	case IndexMethodTypeGiST:
		return "gist"
	case IndexMethodTypeSpGiST:
		return "spgist"
	case IndexMethodTypeGin:
		return "gin"
	case IndexMethodTypeBrin:
		return "brin"
	case IndexMethodTypeIvfflat:
		return "ivfflat"
	default:
		return ""
	}
}

// IndexDef is the struct for index definition.
type IndexDef struct {
	node

	Name    string
	Table   *TableDef
	Unique  bool
	KeyList []*IndexKeyDef
	Method  IndexMethodType
}

// GetKeyNameList to get the name from KeyList.
func (id IndexDef) GetKeyNameList() []string {
	nameList := []string{}
	for _, key := range id.KeyList {
		nameList = append(nameList, key.Key)
	}
	return nameList
}
