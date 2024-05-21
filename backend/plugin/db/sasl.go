package db

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
)

type SASLConfig interface {
	InitEnv() error
	Check() bool
	// used for gohive.
	GetTypeName() string
}

type Realm struct {
	Name                 string
	KDCHost              string
	KDCPort              string
	KDCTransportProtocol string
}
type KerberosConfig struct {
	Primary    string
	Instance   string
	Realm      Realm
	KeytabPath string
}

type KerberosEnv struct {
	krbEnvMutex        *sync.Mutex
	DefaultKrbConfPath string
	realms             map[string]Realm
}

var (
	_            SASLConfig = &KerberosConfig{}
	singletonEnv            = KerberosEnv{
		DefaultKrbConfPath: "/etc/krb5.conf",
	}
	// KinitCmdFmt is the cmd for kinit, should be
	// kinit -kt {keytab file path} {principal}
	// The principal is in {primary}/{instance}@{realm} or {primary}@{realm} format
	// example: 'root/admin@EXAMPLE.COM' or 'root@EXAMPLE.COM'.
	KinitCmdFmt = "kinit -kt %s %s@%s"
	// KrbConfRealmFmt is the config content for /etc/krb5.conf, it should like
	// [realms] {realm} = { kdc = [service/]host[:port]
	// example:
	// [realm]
	//   EXAMPLE.COM = {
	//	   kdc = tcp/192.168.31.2:88
	// 	 }
	KrbConfRealmFmt = "%s = {\n\tkdc = %s/%s:%s\n}"
)

func (krbConfig *KerberosConfig) InitEnv() error {
	if krbConfig.Realm.KDCTransportProtocol != "udp" && krbConfig.Realm.KDCTransportProtocol != "tcp" {
		return errors.Errorf("invalid transport protocol for KDC connection: %s", krbConfig.Realm.KDCTransportProtocol)
	}

	singletonEnv.krbEnvMutex.Lock()
	defer singletonEnv.krbEnvMutex.Unlock()

	if err := singletonEnv.AddRealm(krbConfig.Realm); err != nil {
		return err
	}

	var cmdString string
	if krbConfig.Instance != "" {
		cmdString = fmt.Sprintf(KinitCmdFmt, krbConfig.KeytabPath, krbConfig.Primary, krbConfig.Realm.Name)
	} else {
		primaryAndInstance := fmt.Sprintf("%s/%s", krbConfig.Primary, krbConfig.Instance)
		cmdString = fmt.Sprintf(KinitCmdFmt, krbConfig.KeytabPath, primaryAndInstance, krbConfig.Realm.Name)
	}

	cmd := exec.Command(cmdString)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to execute command kinit")
	}

	return nil
}

func (env *KerberosEnv) AddRealm(realm Realm) error {
	// Sync configurations with local krb5.conf if current realm does not exist.
	_, ok := env.realms[realm.Name]
	if !ok {
		env.realms[realm.Name] = realm

		// Create krb5.cnof file.
		file, err := os.Create(singletonEnv.DefaultKrbConfPath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Write configurations.
		if _, err = file.WriteString("[realms]\n"); err != nil {
			return err
		}
		for _, realm := range env.realms {
			text := fmt.Sprintf(KrbConfRealmFmt, realm.Name, realm.KDCTransportProtocol, realm.KDCHost, realm.KDCPort)
			if _, err = file.WriteString(text); err != nil {
				return err
			}
			if err = file.Sync(); err != nil {
				return err
			}
		}
	}

	return nil
}

// check whether Kerberos is enabled and its settings are valid.
func (krbConfig *KerberosConfig) Check() bool {
	if krbConfig.Primary == "" || krbConfig.Instance == "" || krbConfig.Realm.Name == "" || krbConfig.KeytabPath == "" || krbConfig.Realm.KDCTransportProtocol == "" {
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

func (*PlainSASLConfig) Check() bool {
	return true
}

func (*PlainSASLConfig) GetTypeName() string {
	return "NONE"
}

func (*PlainSASLConfig) InitEnv() error {
	return nil
}
