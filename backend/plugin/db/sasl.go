package db

type SASLConfig interface {
	InitEnv() error
	Check() bool
	// used for gohive.
	GetTypeName() string
}

type KerberosConfig struct {
	Primary  string
	Instance string
	Realm    string
}

var _ SASLConfig = &KerberosConfig{}

// TODO(tommy): use kinit before Kerberos authentication.
func (*KerberosConfig) InitEnv() error {
	return nil
}

// check whether Kerberos is enabled and its settings are valid.
func (config *KerberosConfig) Check() bool {
	if config.Primary == "" || config.Instance == "" || config.Realm == "" {
		return false
	}
	return true
}

func (*KerberosConfig) GetTypeName() string {
	return "KERBEROS"
}

type PlainSASLConfig struct {
	Username string
	Password string
}

var _ SASLConfig = &PlainSASLConfig{}

func (p *PlainSASLConfig) Check() bool {
	return p.Username != ""
}

func (*PlainSASLConfig) GetTypeName() string {
	return "NONE"
}

func (*PlainSASLConfig) InitEnv() error {
	return nil
}
