package db

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		KrbConfPath: "/tmp/krb5.conf",
		KeytabPath:  "/tmp/tmp.keytab",
		// be careful of your environment variables.
		KinitBinPath: "kinit",
		realms:       map[string]Realm{},
		krbEnvMutex:  &sync.Mutex{},
	}

	// The principal is in {primary}/{instance}@{realm} or {primary}@{realm} format, for example:
	// 'root/admin@EXAMPLE.COM' or 'root@EXAMPLE.COM'.
	PrincipalWithoutInstanceFmt = "%s@%s"
	PrincipalWithInstanceFmt    = "%s/%s@%s"
	// KrbConfRealmFmt is the content format for krb5.conf, for example:
	// [realms]
	//   {realm} = {
	//	   kdc = {transport_protocol}/{host}:{port}
	// 	 }
	KrbConfRealmKeyword = "[realms]"
	KrbConfRealmFmt     = "\t%s = {\n\t\tkdc = %s%s:%s\n\t}\n"
	// We have to specify the path of 'krb5.conf' for the 'kinit' command.
	KrbConfEnvVarFmt = "KRB5_CONFIG=%s"
)

// let users manually solve the resource competition problem.
func KrbEnvLock() {
	singletonEnv.krbEnvMutex.Lock()
}

func KrbEnvUnlock() {
	singletonEnv.krbEnvMutex.Unlock()
}

func (krbConfig *KerberosConfig) InitEnv() error {
	// KDCs can use either 'tcp' or 'udp' as their transport protocol.
	if krbConfig.Realm.KDCTransportProtocol != "udp" && krbConfig.Realm.KDCTransportProtocol != "tcp" {
		return errors.Errorf("invalid transport protocol for KDC connection: %s", krbConfig.Realm.KDCTransportProtocol)
	}

	// Create tmp krb5.conf.
	if err := singletonEnv.AddRealm(krbConfig.Realm); err != nil {
		return err
	}

	// Save .keytab file as a temporary file.
	if err := func() error {
		keytabFile, err := os.Create(singletonEnv.KeytabPath)
		if err != nil {
			return err
		}
		defer keytabFile.Close()
		if n, err := keytabFile.Write(krbConfig.Keytab); err != nil || n != len(krbConfig.Keytab) {
			return err
		}
		return keytabFile.Sync()
	}(); err != nil {
		return err
	}

	// kinit -kt {keytab file path} {principal}.
	var cmd *exec.Cmd
	var principal string
	if krbConfig.Instance == "" {
		principal = fmt.Sprintf(PrincipalWithoutInstanceFmt, krbConfig.Primary, krbConfig.Realm.Name)
	} else {
		principal = fmt.Sprintf(PrincipalWithInstanceFmt, krbConfig.Primary, krbConfig.Instance, krbConfig.Realm.Name)
	}
	args := []string{
		fmt.Sprintf(KrbConfEnvVarFmt, singletonEnv.KrbConfPath),
		"kinit",
		"-kt",
		singletonEnv.KeytabPath,
		principal,
	}
	cmd = exec.Command("bash", "-c", strings.Join(args, " "))

	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, fmt.Sprintf("failed to execute command kinit: %s", output))
	}

	return nil
}

func (*KerberosEnv) AddRealm(realm Realm) error {
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

	var kdcConnStr string
	if realm.KDCTransportProtocol == "tcp" {
		// This will force kinit client to communicate with KDC over tcp as Mac
		// doesn't has fall-down mechanism if it fails to communicate over udp.
		kdcConnStr = fmt.Sprintf(KrbConfRealmFmt, realm.Name, "tcp/", realm.KDCHost, realm.KDCPort)
	} else {
		kdcConnStr = fmt.Sprintf(KrbConfRealmFmt, realm.Name, "", realm.KDCHost, realm.KDCPort)
	}
	if _, err = file.WriteString(kdcConnStr); err != nil {
		return err
	}

	return file.Sync()
}

// check whether Kerberos is enabled and its settings are valid.
func (krbConfig *KerberosConfig) Check() bool {
	if krbConfig.Primary == "" || krbConfig.Realm.Name == "" || len(krbConfig.Keytab) == 0 || krbConfig.Realm.KDCTransportProtocol == "" {
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
