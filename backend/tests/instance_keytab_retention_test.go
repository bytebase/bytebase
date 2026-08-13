package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	keytabRetentionHost = "hive.internal.example.com"
	keytabRetentionID   = "admin-ds"
)

// krbDataSource builds a Kerberos data source pointing at host. An empty
// keytab is what a read-modify-write client sends back, since the field is
// INPUT_ONLY and reads return it blank.
func krbDataSource(host, keytab string) *v1pb.DataSource {
	return &v1pb.DataSource{
		Id:       keytabRetentionID,
		Type:     v1pb.DataSourceType_ADMIN,
		Username: "bytebase",
		Host:     host,
		Port:     "10000",
		SaslConfig: &v1pb.SASLConfig{
			Mechanism: &v1pb.SASLConfig_KrbConfig{
				KrbConfig: &v1pb.KerberosConfig{
					Primary: "hive",
					Realm:   "EXAMPLE.COM",
					Keytab:  []byte(keytab),
					KdcHost: "kdc.example.com",
					KdcPort: "88",
				},
			},
		},
	}
}

// createKeytabInstance stands up a Hive instance whose data source carries a
// keytab. Nothing here dials: CreateInstance connects only when validate_only
// is set, so a fabricated host is enough to give a retarget something to
// inherit.
func createKeytabInstance(ctx context.Context, t *testing.T, ctl *controller, instanceID string) *v1pb.Instance {
	t.Helper()
	resp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: instanceID,
		Instance: &v1pb.Instance{
			Title:       instanceID,
			Engine:      v1pb.Engine_HIVE,
			Environment: new("environments/prod"),
			DataSources: []*v1pb.DataSource{krbDataSource(keytabRetentionHost, "stored-keytab")},
		},
	}))
	require.NoError(t, err, "precondition: an instance whose stored keytab a retarget could inherit")
	return resp.Msg
}

// TestKeytabIsNotInheritedByANewDestination pins the retention rule on the
// live paths. A keytab survives an update that omits it — the field is
// INPUT_ONLY, so every read-modify-write client omits it — but it does not
// travel: an update that moves where the data source connects has to supply it
// again.
//
// Both paths that inherit are here. UpdateDataSource merges a partial request
// onto the stored data source; UpdateInstance with update_mask=data_sources
// replaces the list wholesale and matches by data source ID. Before the guard,
// each of the four moving updates below succeeded and the keytab arrived at
// the caller's host.
func TestKeytabIsNotInheritedByANewDestination(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	// Not defer: the parallel subtests below outlive this function body, and a
	// Cleanup runs only once they are all done.
	t.Cleanup(func() { ctl.Close(ctx) })

	t.Run("UpdateDataSource", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		instance := createKeytabInstance(ctx, t, ctl, "keytab-datasource")

		// The destination stands still, so retention works as before: the
		// client edits the realm, omits the keytab, and keeps it.
		sameDestination := krbDataSource(keytabRetentionHost, "")
		sameDestination.SaslConfig.GetKrbConfig().Realm = "OTHER.COM"
		_, err := ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
			Name:       instance.Name,
			DataSource: sameDestination,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"sasl_config"}},
		}))
		a.NoError(err, "an update that connects nowhere new must still inherit the stored keytab")

		// The move. update_mask names the host and the sasl_config that omits
		// the keytab — the read-modify-write shape, one destination field apart.
		_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
			Name:       instance.Name,
			DataSource: krbDataSource("attacker.example.com", ""),
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"host", "sasl_config"}},
		}))
		a.Error(err, "the stored keytab must not follow the data source to a host the caller named")
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
		a.Contains(err.Error(), "keytab")

		// The same move naming ONLY the destination — the shortest version, and
		// the one TestMCPCannotRetargetADataSource sends. The handler patches
		// the stored data source in place, so a mask that never mentions
		// sasl_config leaves the stored keytab sitting on the merged result.
		// Deciding retention on that value would read a stored keytab as a
		// supplied one and let the move through, so what the REQUEST carried
		// is what has to decide.
		destinationOnly := map[string]*v1pb.DataSource{
			"host":     {Id: keytabRetentionID, Host: "attacker.example.com"},
			"port":     {Id: keytabRetentionID, Port: "10001"},
			"ssh_host": {Id: keytabRetentionID, SshHost: "tunnel.attacker.example.com"},
			"ssh_port": {Id: keytabRetentionID, SshPort: "2222"},
			"additional_addresses": {
				Id:                  keytabRetentionID,
				AdditionalAddresses: []*v1pb.DataSource_Address{{Host: "attacker.example.com", Port: "10000"}},
			},
			"extra_connection_parameters": {
				Id:                        keytabRetentionID,
				ExtraConnectionParameters: map[string]string{"host": "attacker.example.com"},
			},
		}
		for path, moved := range destinationOnly {
			_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
				Name:       instance.Name,
				DataSource: moved,
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{path}},
			}))
			a.Error(err, "update_mask=[%q] alone must not carry the stored keytab to the new destination", path)
			a.Contains(err.Error(), "keytab")
		}

		// The mask order must not decide the outcome: sasl_config is applied
		// before host here, so a check inside the mask loop would see the old
		// destination and inherit.
		_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
			Name:       instance.Name,
			DataSource: krbDataSource("attacker.example.com", ""),
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"sasl_config", "host"}},
		}))
		a.Error(err, "the destination is whatever the request ends at, not what the mask reached first")
		a.Contains(err.Error(), "keytab")

		after, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
			Name: instance.Name,
		}))
		a.NoError(err)
		a.Len(after.Msg.DataSources, 1)
		a.Equal(keytabRetentionHost, after.Msg.DataSources[0].Host,
			"a refused move must leave the data source where the operator put it")

		// An update that names neither the destination nor sasl_config must
		// leave the keytab alone. The handler blanks it before the mask loop
		// so that only a supplied one counts, and retention has to put it back
		// — otherwise editing a username would quietly wipe the credential.
		_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
			Name:       instance.Name,
			DataSource: &v1pb.DataSource{Id: keytabRetentionID, Username: "someone-else"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"username"}},
		}))
		a.NoError(err, "an unrelated edit must not need the keytab")

		// The keytab is still there: the guard fires only when there is one to
		// withhold, so a refusal here is what proves the edit above kept it.
		_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
			Name:       instance.Name,
			DataSource: &v1pb.DataSource{Id: keytabRetentionID, Host: "attacker3.example.com"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"host"}},
		}))
		a.Error(err, "the unrelated edit must not have wiped the stored keytab")
		a.Contains(err.Error(), "keytab",
			"it has to be the keytab guard refusing — any other InvalidArgument would keep this "+
				"green with the credential already destroyed")

		// Supplying the keytab again is what the guard asks for, and it is
		// proof the caller holds the credential. The move then goes through.
		_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
			Name:       instance.Name,
			DataSource: krbDataSource("hive2.internal.example.com", "re-supplied-keytab"),
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"host", "sasl_config"}},
		}))
		a.NoError(err, "re-supplying the keytab must let an operator move the instance")

		moved, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
			Name: instance.Name,
		}))
		a.NoError(err)
		a.Equal("hive2.internal.example.com", moved.Msg.DataSources[0].Host)
	})

	t.Run("UpdateInstance full replacement", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		instance := createKeytabInstance(ctx, t, ctl, "keytab-instance")

		replace := func(ds *v1pb.DataSource) error {
			_, err := ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
				Instance:   &v1pb.Instance{Name: instance.Name, DataSources: []*v1pb.DataSource{ds}},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"data_sources"}},
			}))
			return err
		}
		// Every refusal here has to be the keytab guard specifically. The
		// handler can reject a merged data source for several unrelated
		// reasons, and each of these assertions is standing in for "the stored
		// keytab is still there" — a bare a.Error would stay green with the
		// credential already gone.
		refuses := func(ds *v1pb.DataSource, msg string) {
			t.Helper()
			err := replace(ds)
			a.Error(err, msg)
			a.Contains(err.Error(), "keytab", msg)
		}

		// The whole reason retention exists: the list is rebuilt from a read,
		// which never returned the keytab.
		a.NoError(replace(krbDataSource(keytabRetentionHost, "")),
			"a wholesale replacement that stays put must keep the stored keytab")

		refuses(krbDataSource("attacker.example.com", ""),
			"matching by data source ID is not consent to move the keytab")

		// The refusal must not have consumed the stored keytab. The guard
		// fires only when there is one to withhold, so a second refusal is the
		// evidence that it survived.
		refuses(krbDataSource("attacker2.example.com", ""),
			"a refused move must leave the stored keytab in place")

		// The KDC is where kinit presents the keytab itself, so moving it is
		// the most direct version of this move.
		movedKDC := krbDataSource(keytabRetentionHost, "")
		movedKDC.SaslConfig.GetKrbConfig().KdcHost = "kdc.attacker.example.com"
		refuses(movedKDC, "the KDC is the endpoint the keytab material itself reaches")

		a.NoError(replace(krbDataSource("hive2.internal.example.com", "re-supplied-keytab")),
			"re-supplying the keytab must let an operator move the instance")
	})
}
