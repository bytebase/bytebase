package main

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// Testsuite represents the root element of the JUnit XML.
type Testsuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Failures  int        `xml:"failures,attr"`
	Errors    int        `xml:"errors,attr"`
	Testcases []Testcase `xml:"testcase"`
}

// Testcase represents each test case in the JUnit XML.
type Testcase struct {
	File      string   `xml:"file,attr"`
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Failure   *Failure `xml:"failure,omitempty"`
	Error     *Error   `xml:"error,omitempty"`
}

// Failure represents a failed test case.
type Failure struct {
	Text string `xml:",chardata"`
}

// Error represents a test case that resulted in an error.
type Error struct {
	Text string `xml:",chardata"`
}

func writeReleaseCheckToJunitXML(resp *v1pb.CheckReleaseResponse) error {
	root := Testsuite{
		Name: "Bytebase SQL review",
	}

	for _, result := range resp.Results {
		for _, advice := range result.Advices {
			root.Tests++
			testcase := Testcase{
				File:      fmt.Sprintf("%s#L%d", result.File, advice.Line),
				Name:      advice.Title,
				ClassName: fmt.Sprintf("%s,%s", result.File, result.Target),
			}

			switch advice.Status {
			case v1pb.Advice_WARNING:
				root.Failures++
				testcase.Failure = &Failure{
					Text: fmt.Sprintf("[%s] Title: %s\nCode: %d, Line: %d\nContent: %s", advice.Status.String(), advice.Title, advice.Code, advice.Line, advice.Content),
				}
			case v1pb.Advice_ERROR:
				root.Errors++
				testcase.Error = &Error{
					Text: fmt.Sprintf("[%s] Title: %s\nCode: %d, Line: %d\nContent: %s", advice.Status.String(), advice.Title, advice.Code, advice.Line, advice.Content),
				}
			}
			root.Testcases = append(root.Testcases, testcase)
		}
	}
	output, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "error marshaling XML")
	}

	return os.WriteFile("bytebase_junit.xml", []byte(xml.Header+string(output)), 0644)
}
