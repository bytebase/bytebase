// Package configuration provides the parser for mybatis configuration xml file.
package configuration

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// Configuration is the root element of mybatis configuration xml file.
type Configuration struct {
	Environments []Environment
}

// Environment is the element of environments in mybatis configuration xml file.
type Environment struct {
	ID             string
	JDBCConnString string
}

// ParseConfiguration parses the mybatis configuration xml file likes below:
//
// <configuration>
//
//	 <environments default="development">
//	   <environment id="development">
//	     ...
//	   </environment>
//	   <environment id="test">
//	     ...
//	   </environment>
//	</environments>
//
// </configuration>.
func ParseConfiguration(configurationXML string) (*Configuration, error) {
	type Environments struct {
		Environment []struct {
			ID         string `xml:"id,attr"`
			Properties []struct {
				Name  string `xml:"name,attr"`
				Value string `xml:"value,attr"`
			} `xml:"dataSource>property"`
		} `xml:"environment"`
	}

	reader := strings.NewReader(configurationXML)
	d := xml.NewDecoder(reader)
	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "failed to read token")
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "environments" {
				var environments Environments
				if err := d.DecodeElement(&environments, &t); err != nil {
					return nil, errors.Wrapf(err, "failed to decode environments")
				}
				var conf Configuration
				for _, environment := range environments.Environment {
					for _, property := range environment.Properties {
						if property.Name == "url" {
							conf.Environments = append(conf.Environments, Environment{
								ID:             environment.ID,
								JDBCConnString: property.Value,
							})
						}
					}
				}
				return &conf, nil
			}
		default:
		}
	}
}
