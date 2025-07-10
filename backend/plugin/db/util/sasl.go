package util

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// There are two places where the KRB5_CONFIG is needed:
//  1. The 'kinit' command in the subprocess.
//  2. gohive.Connect() in the same process.
func init() {
	if err := os.Setenv("KRB5_CONFIG", tmpKrbConfPath); err != nil {
		panic(fmt.Sprintf("failed to set env %s: %s", "KRB5_CONFIG", tmpKrbConfPath))
	}
}

var (
	Lock = sync.Mutex{}

	// The principal is in {primary}/{instance}@{realm} or {primary}@{realm} format, for example:
	// 'root/admin@EXAMPLE.COM' or 'root@EXAMPLE.COM'.
	principalWithoutInstanceFmt = "%s@%s"
	principalWithInstanceFmt    = "%s/%s@%s"
	// We have to specify the path of 'krb5.conf' for the 'kinit' command.
	tmpKrbConfPath = "/tmp/krb5.conf"
	tmpKeytabPath  = "/tmp/tmp.keytab"
)

func BootKerberosEnv(cfg *storepb.SASLConfig_KrbConfig) error {
	// Create a krb5.conf file.
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
	if cfg.KrbConfig.KdcTransportProtocol == "tcp" && runtime.GOOS == "darwin" {
		protocol = "tcp/"
	}
	cfgContent := fmt.Sprintf(`[libdefaults]
	default_realm = %s
[realms]
	%s = {
		kdc = %s%s:%s
	}
`,
		cfg.KrbConfig.Realm,
		cfg.KrbConfig.Realm,
		protocol, cfg.KrbConfig.KdcHost, cfg.KrbConfig.KdcPort,
	)
	if err := func() error {
		f, err := os.Create(tmpKrbConfPath)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err = f.WriteString(cfgContent); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		return err
	}

	// Create a tmp .keytab.
	if err := func() error {
		f, err := os.Create(tmpKeytabPath)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.Write(cfg.KrbConfig.Keytab); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		return err
	}

	// kinit -kt {keytab file path} {principal}.
	var principal string
	if cfg.KrbConfig.Instance == "" {
		principal = fmt.Sprintf(principalWithoutInstanceFmt, cfg.KrbConfig.Primary, cfg.KrbConfig.Realm)
	} else {
		principal = fmt.Sprintf(principalWithInstanceFmt, cfg.KrbConfig.Primary, cfg.KrbConfig.Instance, cfg.KrbConfig.Realm)
	}
	args := []string{
		"kinit",
		"-kt",
		tmpKeytabPath,
		principal,
	}
	cmd := exec.Command("kinit", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to execute command kinit: %s", output)
	}

	return nil
}
