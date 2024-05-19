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
	Primary  string
	Instance string
	Realm    Realm
	Keytab   []byte
}

type KerberosEnv struct {
	krbEnvMutex  *sync.Mutex
	KrbConfPath  string
	KeytabPath   string
	KinitBinPath string
	realms       map[string]Realm
}

var (
	_            SASLConfig = &KerberosConfig{}
	singletonEnv            = KerberosEnv{
		KrbConfPath:  "/tmp/krb5.conf",
		KeytabPath:   "/tmp/tmp.keytab",
		KinitBinPath: "kinit",
		realms:       map[string]Realm{},
		krbEnvMutex:  &sync.Mutex{},
	}

	// The principal is in {primary}/{instance}@{realm} or {primary}@{realm} format, for example:
	// 'root/admin@EXAMPLE.COM' or 'root@EXAMPLE.COM'.
	PrincipalWithoutInstanceFmt = "%s@%s"
	PrincipalWithInstanceFmt    = "%s/%s@%s"
	// KrbConfRealmFmt is the content format for krb5.conf, for example:
	// [realm]
	//   {realm} = {
	//	   kdc = {transport_protocol}/{host}:{port}
	// 	 }
	KrbConfRealmKeyword = "[realm]"
	KrbConfRealmFmt     = "\t%s = {\n\t\tkdc = %s/%s:%s\n\t}\n"
	// We have to specify the path of 'krb5.conf' for the 'kinit' command.
	KrbConfEnvVarFmt = "KRB5_CONFIG=%s"
)

func (krbConfig *KerberosConfig) InitEnv() error {
	// KDCs can use either 'tcp' or 'udp' as their transport protocol.
	if krbConfig.Realm.KDCTransportProtocol != "udp" && krbConfig.Realm.KDCTransportProtocol != "tcp" {
		return errors.Errorf("invalid transport protocol for KDC connection: %s", krbConfig.Realm.KDCTransportProtocol)
	}

	singletonEnv.krbEnvMutex.Lock()
	defer singletonEnv.krbEnvMutex.Unlock()

	// Create tmp krb5.conf.
	if err := singletonEnv.AddRealm(krbConfig.Realm); err != nil {
		return err
	}

	// Save .keytab file as a temporary file.
	keytabFile, err := os.Create(singletonEnv.KeytabPath)
	if err != nil {
		return err
	}

	// Close and remove the tmp files after the function call.
	// defer os.Remove(singletonEnv.KrbConfPath)
	// defer os.Remove(singletonEnv.KeytabPath)
	defer keytabFile.Close()

	if n, err := keytabFile.Write(krbConfig.Keytab); err != nil || n != len(krbConfig.Keytab) {
		return err
	}
	if err = keytabFile.Sync(); err != nil {
		return err
	}

	// kinit -kt {keytab file path} {principal}.
	var cmd *exec.Cmd
	if krbConfig.Instance == "" {
		principal := fmt.Sprintf(PrincipalWithoutInstanceFmt, krbConfig.Primary, krbConfig.Realm.Name)
		cmd = exec.Command(singletonEnv.KinitBinPath, "-kt", singletonEnv.KeytabPath, principal)
	} else {
		principal := fmt.Sprintf(PrincipalWithInstanceFmt, krbConfig.Primary, krbConfig.Instance, krbConfig.Realm.Name)
		cmd = exec.Command(singletonEnv.KinitBinPath, "-kt", singletonEnv.KeytabPath, principal)
	}

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf(KrbConfEnvVarFmt, singletonEnv.KrbConfPath))

	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, fmt.Sprintf("failed to execute command kinit: %s", output))
	}

	return nil
}

func (env *KerberosEnv) AddRealm(realm Realm) error {
	// Sync configurations with local krb5.conf if current realm does not exist.

	// Create a krb5.conf file.
	file, err := os.Create(singletonEnv.KrbConfPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write configurations.
	if _, err = file.WriteString(fmt.Sprintf("%s\n", KrbConfRealmKeyword)); err != nil {
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
	return nil
}

// check whether Kerberos is enabled and its settings are valid.
func (krbConfig *KerberosConfig) Check() bool {
	if krbConfig.Primary == "" || krbConfig.Instance == "" || krbConfig.Realm.Name == "" || len(krbConfig.Keytab) == 0 || krbConfig.Realm.KDCTransportProtocol == "" {
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
