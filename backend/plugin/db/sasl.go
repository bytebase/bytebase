package db

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// There are two places where the KRB5_CONFIG is needed:
//  1. The 'kinit' command in the subprocess.
//  2. gohive.Connect() in the same process.
func init() {
	if err := os.Setenv("KRB5_CONFIG", DftKrbConfPath); err != nil {
		panic(fmt.Sprintf("failed to set env %s: %s", "KRB5_CONFIG", DftKrbConfPath))
	}
}

type SASLType string

const (
	SASLTypeNone     SASLType = "NONE"
	SASLTypeKerberos SASLType = "KERBEROS"
)

type SASLConfig interface {
	InitEnv() error
	Check() bool
	// used for gohive.
	GetTypeName() SASLType
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
	CurrRealm    *Realm
}

var (
	_            SASLConfig = &KerberosConfig{}
	singletonEnv            = KerberosEnv{
		KrbConfPath: DftKrbConfPath,
		KeytabPath:  DftKeytabPath,
		// be careful of your environment variables.
		KinitBinPath: DftKinitBinPath,
		krbEnvMutex:  &sync.Mutex{},
	}

	// The principal is in {primary}/{instance}@{realm} or {primary}@{realm} format, for example:
	// 'root/admin@EXAMPLE.COM' or 'root@EXAMPLE.COM'.
	PrincipalWithoutInstanceFmt = "%s@%s"
	PrincipalWithInstanceFmt    = "%s/%s@%s"
	// content format of a krb5.conf file:
	// [libdefaults]
	//   default_realm = {realm}
	// [realms]
	//   {realm} = {
	//	   kdc = {transport_protocol}/{host}:{port}
	// 	 }
	KrbConfLibDftKeyword = "[libdefaults]\n"
	KrbConfDftRealmFmt   = "\tdefault_realm = %s\n"
	KrbConfRealmKeyword  = "[realms]\n"
	KrbConfRealmFmt      = "\t%s = {\n\t\tkdc = %s%s:%s\n\t}\n"
	// We have to specify the path of 'krb5.conf' for the 'kinit' command.
	DftKrbConfPath  = "/tmp/krb5.conf"
	DftKeytabPath   = "/tmp/tmp.keytab"
	DftKinitBinPath = "kinit"
)

// let users manually solve the resource competition problem.
func KrbEnvLock() {
	singletonEnv.krbEnvMutex.Lock()
}

func KrbEnvUnlock() {
	singletonEnv.krbEnvMutex.Unlock()
}

func (krbConfig *KerberosConfig) InitEnv() error {
	// Create tmp krb5.conf.
	if err := singletonEnv.SetRealm(krbConfig.Realm); err != nil {
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
		singletonEnv.KinitBinPath,
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

func isRealmSettingsEqual(a, b Realm) bool {
	return a.KDCHost == b.KDCHost && a.KDCTransportProtocol == b.KDCTransportProtocol && a.KDCPort == b.KDCPort && a.Name == b.Name
}

func (e *KerberosEnv) SetRealm(realm Realm) error {
	if e.CurrRealm != nil && isRealmSettingsEqual(*e.CurrRealm, realm) {
		return nil
	}

	// Create a krb5.conf file.
	file, err := os.Create(singletonEnv.KrbConfPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write configurations.
	if _, err = file.WriteString(KrbConfLibDftKeyword); err != nil {
		return err
	}
	if _, err = file.WriteString(fmt.Sprintf(KrbConfDftRealmFmt, realm.Name)); err != nil {
		return err
	}
	if _, err = file.WriteString(KrbConfRealmKeyword); err != nil {
		return err
	}

	var kdcConnStr string
	if realm.KDCTransportProtocol == "tcp" && runtime.GOOS == "darwin" {
		// This will force kinit client to communicate with KDC over tcp as Darwin
		// doesn't has fall-down mechanism if it fails to communicate over udp.
		// However, Linux doesn't need this.
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
	// KDCs can use either 'tcp' or 'udp' as their transport protocol.
	if krbConfig.Realm.KDCTransportProtocol != "udp" && krbConfig.Realm.KDCTransportProtocol != "tcp" {
		return false
	}
	if krbConfig.Primary == "" || krbConfig.Realm.Name == "" || len(krbConfig.Keytab) == 0 || krbConfig.Realm.KDCTransportProtocol == "" {
		return false
	}
	return true
}

func (*KerberosConfig) GetTypeName() SASLType {
	return SASLTypeKerberos
}

type PlainSASLConfig struct {
	Username string
	Password string
}

var _ SASLConfig = &PlainSASLConfig{}

func (*PlainSASLConfig) Check() bool {
	return true
}

func (*PlainSASLConfig) GetTypeName() SASLType {
	return SASLTypeNone
}

func (*PlainSASLConfig) InitEnv() error {
	return nil
}
