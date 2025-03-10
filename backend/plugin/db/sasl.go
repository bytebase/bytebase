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
	if err := os.Setenv("KRB5_CONFIG", dftKrbConfPath); err != nil {
		panic(fmt.Sprintf("failed to set env %s: %s", "KRB5_CONFIG", dftKrbConfPath))
	}
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
	krbConfPath  string
	KinitBinPath string
	CurrRealm    *Realm
}

var (
	lock         = sync.Mutex{}
	singletonEnv = KerberosEnv{
		krbConfPath: dftKrbConfPath,
	}

	// The principal is in {primary}/{instance}@{realm} or {primary}@{realm} format, for example:
	// 'root/admin@EXAMPLE.COM' or 'root@EXAMPLE.COM'.
	principalWithoutInstanceFmt = "%s@%s"
	principalWithInstanceFmt    = "%s/%s@%s"
	// We have to specify the path of 'krb5.conf' for the 'kinit' command.
	dftKrbConfPath = "/tmp/krb5.conf"
	dftKeytabPath  = "/tmp/tmp.keytab"
)

func (krbConfig *KerberosConfig) InitEnv() error {
	lock.Lock()
	defer lock.Unlock()

	// Create tmp krb5.conf.
	if err := singletonEnv.SetRealm(krbConfig.Realm); err != nil {
		return err
	}

	// Save .keytab file as a temporary file.
	if err := func() error {
		keytabFile, err := os.Create(dftKeytabPath)
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
		principal = fmt.Sprintf(principalWithoutInstanceFmt, krbConfig.Primary, krbConfig.Realm.Name)
	} else {
		principal = fmt.Sprintf(principalWithInstanceFmt, krbConfig.Primary, krbConfig.Instance, krbConfig.Realm.Name)
	}
	args := []string{
		"kinit",
		"-kt",
		dftKeytabPath,
		principal,
	}
	cmd = exec.Command("bash", "-c", strings.Join(args, " "))

	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to execute command kinit: %s", output)
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

	// content format of a krb5.conf file:
	// [libdefaults]
	//   default_realm = {realm}
	// [realms]
	//   {realm} = {
	//	   kdc = {transport_protocol}/{host}:{port}
	// 	 }

	protocol := ""
	// This will force kinit client to communicate with KDC over tcp as Darwin
	// doesn't has fall-down mechanism if it fails to communicate over udp.
	// However, Linux doesn't need this.
	if realm.KDCTransportProtocol == "tcp" && runtime.GOOS == "darwin" {
		protocol = "tcp/"
	}

	content := fmt.Sprintf(`[libdefaults]
	default_realm = %s
[realms]
	%s = {
		kdc = %s%s:%s
	}
`,
		realm.Name,
		realm.Name,
		protocol, realm.KDCHost, realm.KDCPort,
	)
	// Create a krb5.conf file.
	f, err := os.Create(singletonEnv.krbConfPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(content); err != nil {
		return err
	}
	return nil
}
