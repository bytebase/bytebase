package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type Conf struct {
	PubKey          string
	Version         string
	Iss             string
	MinimumInstance int
	StorePath       string
}

const (
	keyID           = "v1"
	iss             = "bytebase"
	minimumInstance = 5
)

func NewConf(dataDir string) (*Conf, error) {
	profile := os.Getenv("MODE")
	if profile == "" {
		profile = "dev"
	}

	log.Printf("Get project env %s\n", profile)

	licensePubKey, err := ioutil.ReadFile(fmt.Sprintf("enterprise/keys/%s.pub.pem", profile))
	if err != nil {
		return nil, fmt.Errorf("cannnot read license public key for env %s", profile)
	}

	return &Conf{
		PubKey:          string(licensePubKey),
		Version:         keyID,
		Iss:             iss,
		MinimumInstance: minimumInstance,
		StorePath:       fmt.Sprintf("file:%s/license", dataDir),
	}, nil
}
