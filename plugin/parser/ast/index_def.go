package ast

// IndexDef is the struct for index definition.
type IndexDef struct {
	node

	Name    string
	Table   *TableDef
	Unique  bool
	KeyList []*IndexKeyDef
}

// GetKeyNameList to get the name from KeyList.
func (id IndexDef) GetKeyNameList() []string {
	nameList := []string{}
	for _, key := range id.KeyList {
		nameList = append(nameList, key.Key)
	}
	return nameList
}
