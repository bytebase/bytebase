// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
)

var (
	_ Node = (*TextNode)(nil)
	_ Node = (*DataNode)(nil)
	_ Node = (*ParameterNode)(nil)
	_ Node = (*VariableNode)(nil)
)

// TextNode represents a text node which only contains plain text.
type TextNode struct {
	Text string
}

// RestoreSQL implements Node interface.
func (n *TextNode) RestoreSQL(w io.Writer) error {
	_, err := w.Write([]byte(n.Text))
	return err
}

// ParameterNode represents a parameter node in mybatis mapper xml likes #{param}.
type ParameterNode struct {
	// Name is the name of the parameter.
	Name string
}

// RestoreSQL implements Node interface, parameter node will always be restored to "?".
func (*ParameterNode) RestoreSQL(w io.Writer) error {
	_, err := w.Write([]byte("?"))
	return err
}

// VariableNode represents a variable node in mybatis mapper xml likes ${variable}.
type VariableNode struct {
	// Name is the name of the variable.
	Name string
}

// RestoreSQL implements Node interface, variable node will always be restored to "?".
func (*VariableNode) RestoreSQL(w io.Writer) error {
	_, err := w.Write([]byte("?"))
	return err
}

// DataNode represents a data node which contains plain text, parameter or variable.
type DataNode struct {
	r     *bytes.Reader
	buf   []rune
	Nodes []Node
}

// RestoreSQL implements Node interface.
func (d *DataNode) RestoreSQL(w io.Writer) error {
	for _, node := range d.Nodes {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	return nil
}

// NewDataNode creates a new data node.
func NewDataNode(data []byte) *DataNode {
	return &DataNode{
		r: bytes.NewReader(data),
	}
}

// Scan scans the data node from the bytes given in NewDataNode.
func (d *DataNode) Scan() error {
	if d.r == nil {
		return nil
	}
	defer d.clearBufToTextNode()

	for {
		r, err := d.readRune()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Wrapf(err, "failed to read rune")
		}

		switch r {
		case '#':
			nr, err := d.readRune()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Wrapf(err, "failed to read rune")
			}
			if nr == '{' {
				if err := d.unreadRune(2); err != nil {
					return errors.Wrapf(err, "failed to unread rune")
				}
				d.clearBufToTextNode()
				if err := d.scanParameter(); err != nil {
					return errors.Wrapf(err, "failed to scan parameter")
				}
			}
		case '$':
			nr, err := d.readRune()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Wrapf(err, "failed to read rune")
			}
			if nr == '{' {
				if err := d.unreadRune(2); err != nil {
					return errors.Wrapf(err, "failed to unread rune")
				}
				d.clearBufToTextNode()
				if err := d.scanVariable(); err != nil {
					return errors.Wrapf(err, "failed to scan parameter")
				}
			}
		default:
		}
	}
}

// clearBufToTextNode builds a text node from the buf, clear the buf and append the text node to the nodes.
func (d *DataNode) clearBufToTextNode() {
	if len(d.buf) == 0 {
		return
	}
	d.Nodes = append(d.Nodes, &TextNode{
		Text: string(d.buf),
	})
	d.buf = d.buf[:0]
}

// readRune reads a rune from the reader, and append the rune to the buf.
func (d *DataNode) readRune() (rune, error) {
	r, _, err := d.r.ReadRune()
	if err != nil {
		return 0, err
	}
	d.buf = append(d.buf, r)
	return r, nil
}

func (d *DataNode) unreadRune(size int) error {
	if _, err := d.r.Seek(int64(-size), io.SeekCurrent); err != nil {
		return errors.Wrapf(err, "failed to seek size %d", size)
	}
	d.buf = d.buf[:len(d.buf)-size]
	return nil
}

// scanParameter scans the parameter node from the reader, likes #{param}.
func (d *DataNode) scanParameter() error {
	// skip the first "#{".
	if _, err := d.readRune(); err != nil {
		return errors.Wrapf(err, "failed to read rune")
	}
	if _, err := d.readRune(); err != nil {
		return errors.Wrapf(err, "failed to read rune")
	}

	for {
		r, err := d.readRune()
		if err != nil {
			if err == io.EOF {
				return errors.Wrapf(err, "expected read rune '}' to close parameter node, but meet EOF")
			}
			return errors.Wrapf(err, "failed to read rune")
		}
		if r == '}' {
			partBuf := string(d.buf[ /*Skip '#{'*/ 2 : len(d.buf)-1 /*Skip '}'*/])
			d.Nodes = append(d.Nodes, &ParameterNode{
				Name: string(partBuf),
			})
			d.buf = d.buf[:0]
			return nil
		}
	}
}

// scanVariable scans the variable node from the reader, likes ${variable}.
func (d *DataNode) scanVariable() error {
	// skip the first "${".
	if _, err := d.readRune(); err != nil {
		return errors.Wrapf(err, "failed to read rune")
	}
	if _, err := d.readRune(); err != nil {
		return errors.Wrapf(err, "failed to read rune")
	}

	for {
		r, err := d.readRune()
		if err != nil {
			if err == io.EOF {
				return errors.Wrapf(err, "expected read rune '}' to close parameter node, but meet EOF")
			}
			return errors.Wrapf(err, "failed to read rune")
		}
		if r == '}' {
			partBuf := string(d.buf[2: /*Skip '${'*/])
			d.Nodes = append(d.Nodes, &VariableNode{
				Name: string(partBuf),
			})
			d.buf = d.buf[:0]
		}
	}
}
