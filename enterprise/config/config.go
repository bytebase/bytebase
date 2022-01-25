package config

import (
	"fmt"
	"io/ioutil"

	"go.uber.org/zap"
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

func NewConf(l *zap.Logger, dataDir string, mode string) (*Conf, error) {
	l.Info("get project env", zap.String("env", mode))

	licensePubKey, err := ioutil.ReadFile(fmt.Sprintf("enterprise/keys/%s.pub.pem", mode))
	if err != nil {
		return nil, fmt.Errorf("cannot read license public key for env %s", mode)
	}

	return &Conf{
		PubKey:          string(licensePubKey),
		Version:         keyID,
		Iss:             iss,
		MinimumInstance: minimumInstance,
		StorePath:       fmt.Sprintf("file:%s/license_%s", dataDir, mode),
	}, nil
}
