// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"encoding/xml"
	"io"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	_ Node = (*IfNode)(nil)

	_ Node = (*ChooseNode)(nil)
	_ Node = (*WhenNode)(nil)
	_ Node = (*OtherwiseNode)(nil)

	_ Node = (*WhereNode)(nil)
	_ Node = (*SetNode)(nil)
	_ Node = (*TrimNode)(nil)
	_ Node = (*ForEachNode)(nil)

	_ Node = (*SQLNode)(nil)
	_ Node = (*IncludeNode)(nil)
	_ Node = (*PropertyNode)(nil)
)

// IfNode represents a if node in mybatis mapper xml likes <if test="condition">...</if>.
type IfNode struct {
	Test     string
	Children []Node
}

// NewIfNode creates a new if node.
func NewIfNode(startElement *xml.StartElement) *IfNode {
	node := &IfNode{}
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "test" {
			node.Test = attr.Value
		}
	}
	return node
}

// RestoreSQL implements Node interface, the if condition will be ignored.
func (n *IfNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (*IfNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L290
	switch child.(type) {
	case *DataNode, *IncludeNode, *TrimNode, *WhereNode, *SetNode, *ForEachNode, *ChooseNode, *IfNode:
		return true
	default:
		return false
	}
}

// AddChild adds a child to the if node.
func (n *IfNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

// ChooseNode represents a choose node in mybatis mapper xml likes <choose>...</choose>.
type ChooseNode struct {
	Children []Node
}

// NewChooseNode creates a new choose node.
func NewChooseNode(_ *xml.StartElement) *ChooseNode {
	return &ChooseNode{}
}

// RestoreSQL implements Node interface.
func (n *ChooseNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (*ChooseNode) isChildAcceptable(child Node) bool {
	switch child.(type) {
	case *WhenNode, *OtherwiseNode:
		return true
	default:
		return false
	}
}

// AddChild implements Node interface.
func (n *ChooseNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

// WhenNode represents a when node in mybatis mapper xml select node likes <select><when test="condition">...</when></select>.
type WhenNode struct {
	Test     string
	Children []Node
}

// NewWhenNode creates a new when node.
func NewWhenNode(startElement *xml.StartElement) *WhenNode {
	node := &WhenNode{}
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "test" {
			node.Test = attr.Value
		}
	}
	return node
}

// RestoreSQL implements Node interface, the when condition will be ignored.
func (n *WhenNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (*WhenNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#LL284C1-L284C1
	switch child.(type) {
	case *DataNode, *IncludeNode, *TrimNode, *WhereNode, *SetNode, *ForEachNode, *ChooseNode, *IfNode:
		return true
	default:
		return false
	}
}

// AddChild adds a child to the when node.
func (n *WhenNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

// OtherwiseNode represents a otherwise node in mybatis mapper xml select node likes <select><otherwise>...</otherwise></select>.
type OtherwiseNode struct {
	Children []Node
}

// NewOtherwiseNode creates a new otherwise node.
func NewOtherwiseNode(_ *xml.StartElement) *OtherwiseNode {
	return &OtherwiseNode{}
}

// RestoreSQL implements Node interface.
func (n *OtherwiseNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (*OtherwiseNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L288
	switch child.(type) {
	case *DataNode, *IncludeNode, *TrimNode, *WhereNode, *SetNode, *ForEachNode, *ChooseNode, *IfNode:
		return true
	default:
		return false
	}
}

// AddChild adds a child to the otherwise node.
func (n *OtherwiseNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

// TrimNode represents a trim node in mybatis mapper xml likes <trim prefix="prefix" suffix="suffix" prefixOverrides="prefixOverrides" suffixOverrides="suffixOverrides">...</trim>.
type TrimNode struct {
	Prefix               string
	Suffix               string
	PrefixOverridesParts []string
	SuffixOverridesParts []string
	Children             []Node
}

// NewTrimNode creates a new trim node.
func NewTrimNode(startElement *xml.StartElement) *TrimNode {
	var prefix, suffix, prefixOverrides, suffixOverrides string
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "prefix":
			prefix = attr.Value
		case "suffix":
			suffix = attr.Value
		case "prefixOverrides":
			prefixOverrides = attr.Value
		case "suffixOverrides":
			suffixOverrides = attr.Value
		}
	}
	return newTrimNodeWithAttrs(prefix, suffix, prefixOverrides, suffixOverrides)
}

// newTrimNodeWithAttrs creates a new trim node with given attributes.
func newTrimNodeWithAttrs(prefix, suffix, prefixOverrides, suffixOverrides string) *TrimNode {
	prefixOverridesParts := strings.Split(prefixOverrides, "|")
	suffixOverridesParts := strings.Split(suffixOverrides, "|")
	return &TrimNode{
		Prefix:               prefix,
		Suffix:               suffix,
		PrefixOverridesParts: prefixOverridesParts,
		SuffixOverridesParts: suffixOverridesParts,
	}
}

// RestoreSQL implements Node interface.
func (n *TrimNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	var stringsBuilder strings.Builder
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, &stringsBuilder); err != nil {
			return err
		}
	}
	trimmed := strings.TrimSpace(stringsBuilder.String())
	if len(trimmed) == 0 {
		return nil
	}
	// Replace the prefix and suffix with empty string if matches the part in prefixOverridesParts and suffixOverridesParts.
	for _, part := range n.PrefixOverridesParts {
		if strings.HasPrefix(trimmed, part) {
			trimmed = strings.TrimPrefix(trimmed, part)
			break
		}
	}
	for _, part := range n.SuffixOverridesParts {
		if strings.HasSuffix(trimmed, part) {
			trimmed = strings.TrimSuffix(trimmed, part)
			break
		}
	}
	if len(n.Prefix) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Prefix)); err != nil {
			return err
		}
	}
	if len(trimmed) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(trimmed)); err != nil {
			return err
		}
	}
	if len(n.Suffix) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Suffix)); err != nil {
			return err
		}
	}
	return nil
}

func (*TrimNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L262
	switch child.(type) {
	case *DataNode, *IncludeNode, *TrimNode, *WhereNode, *SetNode, *ForEachNode, *ChooseNode, *IfNode:
		return true
	default:
		return false
	}
}

// AddChild adds a child to the trim node.
func (n *TrimNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

// WhereNode represents a where node in mybatis mapper xml likes <where>...</where>.
type WhereNode struct {
	trimNode *TrimNode
}

// NewWhereNode creates a new where node.
func NewWhereNode(_ *xml.StartElement) *WhereNode {
	return &WhereNode{
		trimNode: newTrimNodeWithAttrs("WHERE", "", "AND |OR ", ""),
	}
}

// RestoreSQL implements Node interface.
func (n *WhereNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	return n.trimNode.RestoreSQL(ctx, w)
}

func (n *WhereNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L269
	return n.trimNode.isChildAcceptable(child)
}

// AddChild adds a child to the where node.
func (n *WhereNode) AddChild(child Node) {
	n.trimNode.AddChild(child)
}

// SetNode represents a set node in mybatis mapper xml likes <set>...</set>.
type SetNode struct {
	trimNode *TrimNode
}

// NewSetNode creates a new set node.
func NewSetNode(_ *xml.StartElement) *SetNode {
	return &SetNode{
		trimNode: newTrimNodeWithAttrs("SET", "", "", ","),
	}
}

// RestoreSQL implements Node interface.
func (n *SetNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	return n.trimNode.RestoreSQL(ctx, w)
}

func (n *SetNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L270
	return n.trimNode.isChildAcceptable(child)
}

// AddChild adds a child to the set node.
func (n *SetNode) AddChild(child Node) {
	n.trimNode.AddChild(child)
}

// ForEachNode represents a foreach node in mybatis mapper xml likes <foreach collection="collection" item="item" index="index" open="open" close="close" separator="separator">...</foreach>.
type ForEachNode struct {
	Collection string
	Item       string
	Separator  string
	Open       string
	Close      string
	Index      string
	Children   []Node
}

// NewForeachNode creates a new foreach node.
func NewForeachNode(startElement *xml.StartElement) *ForEachNode {
	var eachNode ForEachNode
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "collection":
			eachNode.Collection = attr.Value
		case "item":
			eachNode.Item = attr.Value
		case "index":
			eachNode.Index = attr.Value
		case "open":
			eachNode.Open = attr.Value
		case "close":
			eachNode.Close = attr.Value
		case "separator":
			eachNode.Separator = attr.Value
		}
	}
	return &eachNode
}

func (*ForEachNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L272
	switch child.(type) {
	case *DataNode, *IncludeNode, *TrimNode, *WhereNode, *SetNode, *ForEachNode, *ChooseNode, *IfNode:
		return true
	default:
		return false
	}
}

// AddChild adds a child to the foreach node.
func (n *ForEachNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

// RestoreSQL implements Node interface.
func (n *ForEachNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	var partBuilder strings.Builder
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, &partBuilder); err != nil {
			return err
		}
	}
	part := strings.TrimSpace(partBuilder.String())
	if len(part) == 0 {
		return nil
	}
	if _, err := w.Write([]byte(" ")); err != nil {
		return err
	}
	if len(n.Open) > 0 {
		if _, err := w.Write([]byte(n.Open)); err != nil {
			return err
		}
	}
	// We write the part twice.
	if _, err := w.Write([]byte(part)); err != nil {
		return err
	}
	if len(n.Separator) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Separator)); err != nil {
			return err
		}
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(part)); err != nil {
			return err
		}
	}
	if len(n.Close) > 0 {
		if _, err := w.Write([]byte(n.Close)); err != nil {
			return err
		}
	}
	return nil
}

// SQLNode represents a sql node in mybatis mapper xml likes <sql id="sqlId">...</sql>.
type SQLNode struct {
	ID       string
	Children []Node
}

// NewSQLNode creates a new sql node.
func NewSQLNode(startElement *xml.StartElement) *SQLNode {
	var id string
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "id" {
			id = attr.Value
			break
		}
	}
	return &SQLNode{
		ID: id,
	}
}

func (*SQLNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L255
	switch child.(type) {
	case *DataNode, *IncludeNode, *TrimNode, *WhereNode, *SetNode, *ForEachNode, *IfNode, *ChooseNode:
		return true
	default:
		return false
	}
}

// AddChild adds a child to the sql node.
func (n *SQLNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

// RestoreSQL implements Node interface.
// SQLNode is a placeholder node, it will be replaced by the sql node with the same id.
// So we don't need to restore the sql node.
func (*SQLNode) RestoreSQL(*RestoreContext, io.Writer) error {
	return nil
}

func (n *SQLNode) String(ctx *RestoreContext) (string, error) {
	var builder strings.Builder
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, &builder); err != nil {
			return "", err
		}
	}
	return builder.String(), nil
}

// IncludeNode represents a include node in mybatis mapper xml likes <include refid="refId">...</include>.
type IncludeNode struct {
	RefID string
	// IncludeNode can only contains property node.
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L244
	PropertyChildren []*PropertyNode
}

// NewIncludeNode creates a new include node.
func NewIncludeNode(startElement *xml.StartElement) *IncludeNode {
	var refID string
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "refid" {
			refID = attr.Value
			break
		}
	}
	return &IncludeNode{
		RefID: refID,
	}
}

// AddChild adds a child to the include node.
func (n *IncludeNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.PropertyChildren = append(n.PropertyChildren, child.(*PropertyNode))
}

func (*IncludeNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L244
	if _, ok := child.(*PropertyNode); ok {
		return true
	}
	return false
}

// RestoreSQL implements Node interface.
func (n *IncludeNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	variableCatcher := regexp.MustCompile(`\${([a-zA-Z0-9_]+)}`)
	// We need to replace the variable in the refID.
	refID := variableCatcher.ReplaceAllStringFunc(n.RefID, func(s string) string {
		matches := variableCatcher.FindStringSubmatch(s)
		if len(matches) != 2 {
			return s
		}
		name := matches[1]
		return ctx.Variable[name]
	})

	sqlNode, ok := ctx.SQLMap[refID]
	if !ok {
		return errors.Errorf("refID %s not found", n.RefID)
	}

	// Set all the properties.
	// It is safe we don't check whether the variable exists, because property element can only be child of include element,
	// and include element can only refers one sql element. The property element was covered by another property element
	// is not a problem, because the outer include element will not use the outer property element any more.
	for _, propertyNode := range n.PropertyChildren {
		ctx.Variable[propertyNode.Name] = propertyNode.Value
	}

	sqlString, err := sqlNode.String(ctx)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(sqlString)
	if len(trimmed) == 0 {
		return nil
	}
	if _, err := w.Write([]byte(" ")); err != nil {
		return err
	}
	if _, err := w.Write([]byte(trimmed)); err != nil {
		return err
	}

	// Unset all the properties.
	for _, propertyNode := range n.PropertyChildren {
		delete(ctx.Variable, propertyNode.Name)
	}

	return nil
}

// PropertyNode represents a property node in mybatis mapper xml likes <property name="name" value="value" />.
type PropertyNode struct {
	Name  string
	Value string
}

// NewPropertyNode creates a new property node.
func NewPropertyNode(startElement *xml.StartElement) *PropertyNode {
	var name, value string
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "name" {
			name = attr.Value
		} else if attr.Name.Local == "value" {
			value = attr.Value
		}
	}
	return &PropertyNode{
		Name:  name,
		Value: value,
	}
}

func (*PropertyNode) isChildAcceptable(Node) bool {
	return false
}

// AddChild adds a child to the property node.
func (*PropertyNode) AddChild(Node) {
}

// RestoreSQL implements Node interface.
func (*PropertyNode) RestoreSQL(*RestoreContext, io.Writer) error {
	return nil
}
