package v1

import (
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func convertInstanceMessage(instance *store.InstanceMessage) *v1pb.Instance {
	engine := convertToEngine(instance.Metadata.GetEngine())
	dataSources := convertDataSources(instance.Metadata.GetDataSources())

	return &v1pb.Instance{
		Name:          buildInstanceName(instance.ResourceID),
		Title:         instance.Metadata.GetTitle(),
		Engine:        engine,
		EngineVersion: instance.Metadata.GetVersion(),
		ExternalLink:  instance.Metadata.GetExternalLink(),
		DataSources:   dataSources,
		State:         convertDeletedToState(instance.Deleted),
		Environment:   buildEnvironmentName(instance.EnvironmentID),
		Activation:    instance.Metadata.GetActivation(),
		SyncInterval:  instance.Metadata.GetSyncInterval(),
		SyncDatabases: instance.Metadata.GetSyncDatabases(),
		Roles:         convertInstanceRoles(instance, instance.Metadata.GetRoles()),
		LastSyncTime:  instance.Metadata.GetLastSyncTime(),
		Labels:        instance.Metadata.GetLabels(),
	}
}

// buildRoleName builds the role name with the given instance ID and role name.
func buildRoleName(b *strings.Builder, instanceID, roleName string) string {
	b.Reset()
	_, _ = b.WriteString(common.InstanceNamePrefix)
	_, _ = b.WriteString(instanceID)
	_, _ = b.WriteString("/")
	_, _ = b.WriteString(common.RolePrefix)
	_, _ = b.WriteString(roleName)
	return b.String()
}

func convertInstanceRoles(instance *store.InstanceMessage, roles []*storepb.InstanceRole) []*v1pb.InstanceRole {
	var v1Roles []*v1pb.InstanceRole
	var b strings.Builder

	// preallocate memory for the builder
	b.Grow(len(common.InstanceNamePrefix) + len(instance.ResourceID) + 1 + len(common.RolePrefix) + 20)

	for _, role := range roles {
		v1Roles = append(v1Roles, &v1pb.InstanceRole{
			Name:      buildRoleName(&b, instance.ResourceID, role.Name),
			RoleName:  role.Name,
			Attribute: role.Attribute,
		})
	}
	return v1Roles
}

func convertInstanceToInstanceMessage(instanceID string, instance *v1pb.Instance) (*store.InstanceMessage, error) {
	datasources, err := convertV1DataSources(instance.DataSources)
	if err != nil {
		return nil, err
	}

	var environmentID *string
	if instance.Environment != nil && *instance.Environment != "" {
		envID, err := common.GetEnvironmentID(*instance.Environment)
		if err != nil {
			return nil, err
		}
		environmentID = &envID
	}

	return &store.InstanceMessage{
		ResourceID:    instanceID,
		EnvironmentID: environmentID,
		Metadata: &storepb.Instance{
			Title:         instance.GetTitle(),
			Engine:        convertEngine(instance.Engine),
			ExternalLink:  instance.GetExternalLink(),
			Activation:    instance.GetActivation(),
			DataSources:   datasources,
			SyncInterval:  instance.GetSyncInterval(),
			SyncDatabases: instance.GetSyncDatabases(),
			Labels:        instance.GetLabels(),
		},
	}, nil
}

func convertInstanceMessageToInstanceResource(instanceMessage *store.InstanceMessage) *v1pb.InstanceResource {
	instance := convertInstanceMessage(instanceMessage)
	return &v1pb.InstanceResource{
		Name:          instance.Name,
		Title:         instance.Title,
		Engine:        instance.Engine,
		EngineVersion: instance.EngineVersion,
		DataSources:   instance.DataSources,
		Activation:    instance.Activation,
		Environment:   instance.Environment,
	}
}

func convertV1DataSources(dataSources []*v1pb.DataSource) ([]*storepb.DataSource, error) {
	var values []*storepb.DataSource
	for _, ds := range dataSources {
		dataSource, err := convertV1DataSource(ds)
		if err != nil {
			return nil, err
		}
		values = append(values, dataSource)
	}

	return values, nil
}

func convertDataSourceExternalSecret(externalSecret *storepb.DataSourceExternalSecret) *v1pb.DataSourceExternalSecret {
	if externalSecret == nil {
		return nil
	}

	resp := &v1pb.DataSourceExternalSecret{
		SecretType:               v1pb.DataSourceExternalSecret_SecretType(externalSecret.SecretType),
		Url:                      externalSecret.Url,
		AuthType:                 v1pb.DataSourceExternalSecret_AuthType(externalSecret.AuthType),
		EngineName:               externalSecret.EngineName,
		SecretName:               externalSecret.SecretName,
		PasswordKeyName:          externalSecret.PasswordKeyName,
		SkipVaultTlsVerification: externalSecret.SkipVaultTlsVerification,
		// Clear sensitive Vault SSL data (INPUT_ONLY fields, should not be returned)
		VaultSslCa:   "",
		VaultSslCert: "",
		VaultSslKey:  "",
	}

	// clear sensitive data.
	switch resp.AuthType {
	case v1pb.DataSourceExternalSecret_VAULT_APP_ROLE:
		appRole := externalSecret.GetAppRole()
		if appRole != nil {
			resp.AuthOption = &v1pb.DataSourceExternalSecret_AppRole{
				AppRole: &v1pb.DataSourceExternalSecret_AppRoleAuthOption{
					Type:      v1pb.DataSourceExternalSecret_AppRoleAuthOption_SecretType(appRole.Type),
					MountPath: appRole.MountPath,
				},
			}
		}
	case v1pb.DataSourceExternalSecret_TOKEN:
		resp.AuthOption = &v1pb.DataSourceExternalSecret_Token{
			Token: "",
		}
	default:
	}

	return resp
}

func convertDataSources(dataSources []*storepb.DataSource) []*v1pb.DataSource {
	var v1DataSources []*v1pb.DataSource
	for _, ds := range dataSources {
		externalSecret := convertDataSourceExternalSecret(ds.GetExternalSecret())

		dataSourceType := v1pb.DataSourceType_DATA_SOURCE_UNSPECIFIED
		switch ds.GetType() {
		case storepb.DataSourceType_ADMIN:
			dataSourceType = v1pb.DataSourceType_ADMIN
		case storepb.DataSourceType_READ_ONLY:
			dataSourceType = v1pb.DataSourceType_READ_ONLY
		default:
		}

		authenticationType := v1pb.DataSource_AUTHENTICATION_UNSPECIFIED
		switch ds.GetAuthenticationType() {
		case storepb.DataSource_AUTHENTICATION_UNSPECIFIED, storepb.DataSource_PASSWORD:
			authenticationType = v1pb.DataSource_PASSWORD
		case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
			authenticationType = v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM
		case storepb.DataSource_AWS_RDS_IAM:
			authenticationType = v1pb.DataSource_AWS_RDS_IAM
		case storepb.DataSource_AZURE_IAM:
			authenticationType = v1pb.DataSource_AZURE_IAM
		default:
		}

		dataSource := &v1pb.DataSource{
			Id:       ds.GetId(),
			Type:     dataSourceType,
			Username: ds.GetUsername(),
			// We don't return the password and SSLs on reads.
			Host:                      ds.GetHost(),
			Port:                      ds.GetPort(),
			Database:                  ds.GetDatabase(),
			Srv:                       ds.GetSrv(),
			AuthenticationDatabase:    ds.GetAuthenticationDatabase(),
			Sid:                       ds.GetSid(),
			ServiceName:               ds.GetServiceName(),
			SshHost:                   ds.GetSshHost(),
			SshPort:                   ds.GetSshPort(),
			SshUser:                   ds.GetSshUser(),
			ExternalSecret:            externalSecret,
			AuthenticationType:        authenticationType,
			SaslConfig:                convertDataSourceSaslConfig(ds.GetSaslConfig()),
			AdditionalAddresses:       convertDataSourceAddresses(ds.GetAdditionalAddresses()),
			ReplicaSet:                ds.GetReplicaSet(),
			DirectConnection:          ds.GetDirectConnection(),
			Region:                    ds.GetRegion(),
			WarehouseId:               ds.GetWarehouseId(),
			UseSsl:                    ds.GetUseSsl(),
			VerifyTlsCertificate:      ds.GetVerifyTlsCertificate(),
			RedisType:                 convertRedisType(ds.GetRedisType()),
			MasterName:                ds.GetMasterName(),
			MasterUsername:            ds.GetMasterUsername(),
			ExtraConnectionParameters: ds.GetExtraConnectionParameters(),
		}

		switch dataSource.AuthenticationType {
		case v1pb.DataSource_AZURE_IAM:
			if azureCredential := ds.GetAzureCredential(); azureCredential != nil {
				dataSource.IamExtension = &v1pb.DataSource_AzureCredential_{
					AzureCredential: &v1pb.DataSource_AzureCredential{
						TenantId: azureCredential.TenantId,
						ClientId: azureCredential.ClientId,
					},
				}
			}
		case v1pb.DataSource_AWS_RDS_IAM:
			if awsCredential := ds.GetAwsCredential(); awsCredential != nil {
				dataSource.IamExtension = &v1pb.DataSource_AwsCredential{
					AwsCredential: &v1pb.DataSource_AWSCredential{
						RoleArn:    awsCredential.RoleArn,
						ExternalId: awsCredential.ExternalId,
					},
				}
			}
		case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
			if gcpCredential := ds.GetGcpCredential(); gcpCredential != nil {
				dataSource.IamExtension = &v1pb.DataSource_GcpCredential{
					GcpCredential: &v1pb.DataSource_GCPCredential{},
				}
			}
		default:
		}

		v1DataSources = append(v1DataSources, dataSource)
	}

	return v1DataSources
}

func convertV1DataSourceExternalSecret(externalSecret *v1pb.DataSourceExternalSecret) (*storepb.DataSourceExternalSecret, error) {
	if externalSecret == nil {
		return nil, nil
	}

	secret := &storepb.DataSourceExternalSecret{
		SecretType:               storepb.DataSourceExternalSecret_SecretType(externalSecret.SecretType),
		Url:                      externalSecret.Url,
		AuthType:                 storepb.DataSourceExternalSecret_AuthType(externalSecret.AuthType),
		EngineName:               externalSecret.EngineName,
		SecretName:               externalSecret.SecretName,
		PasswordKeyName:          externalSecret.PasswordKeyName,
		SkipVaultTlsVerification: externalSecret.SkipVaultTlsVerification,
		VaultSslCa:               externalSecret.VaultSslCa,
		VaultSslCert:             externalSecret.VaultSslCert,
		VaultSslKey:              externalSecret.VaultSslKey,
	}

	// Convert auth options
	switch externalSecret.AuthOption.(type) {
	case *v1pb.DataSourceExternalSecret_Token:
		secret.AuthOption = &storepb.DataSourceExternalSecret_Token{
			Token: externalSecret.GetToken(),
		}
	case *v1pb.DataSourceExternalSecret_AppRole:
		appRole := externalSecret.GetAppRole()
		if appRole != nil {
			secret.AuthOption = &storepb.DataSourceExternalSecret_AppRole{
				AppRole: &storepb.DataSourceExternalSecret_AppRoleAuthOption{
					Type:      storepb.DataSourceExternalSecret_AppRoleAuthOption_SecretType(appRole.Type),
					MountPath: appRole.MountPath,
					RoleId:    appRole.RoleId,
					SecretId:  appRole.SecretId,
				},
			}
		}
	}

	switch secret.SecretType {
	case storepb.DataSourceExternalSecret_VAULT_KV_V2:
		if secret.Url == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Vault URL"))
		}
		if secret.EngineName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Vault engine name"))
		}
		if secret.SecretName == "" || secret.PasswordKeyName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing secret name or key name"))
		}
	case storepb.DataSourceExternalSecret_AWS_SECRETS_MANAGER:
		if secret.SecretName == "" || secret.PasswordKeyName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing secret name or key name"))
		}
	case storepb.DataSourceExternalSecret_GCP_SECRET_MANAGER:
		if secret.SecretName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing GCP secret name"))
		}
	case storepb.DataSourceExternalSecret_AZURE_KEY_VAULT:
		if secret.Url == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Azure Key Vault URL"))
		}
		if secret.SecretName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Azure Key Vault secret name"))
		}
	default:
	}

	switch secret.AuthType {
	case storepb.DataSourceExternalSecret_TOKEN:
		if secret.GetToken() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing token"))
		}
	case storepb.DataSourceExternalSecret_VAULT_APP_ROLE:
		if secret.GetAppRole() == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Vault approle"))
		}
	default:
	}

	return secret, nil
}

func convertV1DataSourceSaslConfig(saslConfig *v1pb.SASLConfig) *storepb.SASLConfig {
	if saslConfig == nil {
		return nil
	}
	storeSaslConfig := &storepb.SASLConfig{}
	switch m := saslConfig.Mechanism.(type) {
	case *v1pb.SASLConfig_KrbConfig:
		storeSaslConfig.Mechanism = &storepb.SASLConfig_KrbConfig{
			KrbConfig: &storepb.KerberosConfig{
				Primary:              m.KrbConfig.Primary,
				Instance:             m.KrbConfig.Instance,
				Realm:                m.KrbConfig.Realm,
				Keytab:               m.KrbConfig.Keytab,
				KdcHost:              m.KrbConfig.KdcHost,
				KdcPort:              m.KrbConfig.KdcPort,
				KdcTransportProtocol: m.KrbConfig.KdcTransportProtocol,
			},
		}
	default:
		return nil
	}
	return storeSaslConfig
}

func convertDataSourceSaslConfig(saslConfig *storepb.SASLConfig) *v1pb.SASLConfig {
	if saslConfig == nil {
		return nil
	}
	storeSaslConfig := &v1pb.SASLConfig{}
	switch m := saslConfig.Mechanism.(type) {
	case *storepb.SASLConfig_KrbConfig:
		storeSaslConfig.Mechanism = &v1pb.SASLConfig_KrbConfig{
			KrbConfig: &v1pb.KerberosConfig{
				Primary:              m.KrbConfig.Primary,
				Instance:             m.KrbConfig.Instance,
				Realm:                m.KrbConfig.Realm,
				Keytab:               m.KrbConfig.Keytab,
				KdcHost:              m.KrbConfig.KdcHost,
				KdcPort:              m.KrbConfig.KdcPort,
				KdcTransportProtocol: m.KrbConfig.KdcTransportProtocol,
			},
		}
	default:
		return nil
	}
	return storeSaslConfig
}

func convertDataSourceAddresses(addresses []*storepb.DataSource_Address) []*v1pb.DataSource_Address {
	res := make([]*v1pb.DataSource_Address, 0, len(addresses))
	for _, address := range addresses {
		res = append(res, &v1pb.DataSource_Address{
			Host: address.Host,
			Port: address.Port,
		})
	}
	return res
}

func convertAdditionalAddresses(addresses []*v1pb.DataSource_Address) []*storepb.DataSource_Address {
	res := make([]*storepb.DataSource_Address, 0, len(addresses))
	for _, address := range addresses {
		res = append(res, &storepb.DataSource_Address{
			Host: address.Host,
			Port: address.Port,
		})
	}
	return res
}

func convertV1AuthenticationType(authType v1pb.DataSource_AuthenticationType) storepb.DataSource_AuthenticationType {
	authenticationType := storepb.DataSource_AUTHENTICATION_UNSPECIFIED
	switch authType {
	case v1pb.DataSource_AUTHENTICATION_UNSPECIFIED, v1pb.DataSource_PASSWORD:
		authenticationType = storepb.DataSource_PASSWORD
	case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		authenticationType = storepb.DataSource_GOOGLE_CLOUD_SQL_IAM
	case v1pb.DataSource_AWS_RDS_IAM:
		authenticationType = storepb.DataSource_AWS_RDS_IAM
	case v1pb.DataSource_AZURE_IAM:
		authenticationType = storepb.DataSource_AZURE_IAM
	default:
	}
	return authenticationType
}

func convertV1RedisType(redisType v1pb.DataSource_RedisType) storepb.DataSource_RedisType {
	authenticationType := storepb.DataSource_REDIS_TYPE_UNSPECIFIED
	switch redisType {
	case v1pb.DataSource_STANDALONE:
		authenticationType = storepb.DataSource_STANDALONE
	case v1pb.DataSource_SENTINEL:
		authenticationType = storepb.DataSource_SENTINEL
	case v1pb.DataSource_CLUSTER:
		authenticationType = storepb.DataSource_CLUSTER
	default:
	}
	return authenticationType
}

func convertRedisType(redisType storepb.DataSource_RedisType) v1pb.DataSource_RedisType {
	authenticationType := v1pb.DataSource_STANDALONE
	switch redisType {
	case storepb.DataSource_STANDALONE:
		authenticationType = v1pb.DataSource_STANDALONE
	case storepb.DataSource_SENTINEL:
		authenticationType = v1pb.DataSource_SENTINEL
	case storepb.DataSource_CLUSTER:
		authenticationType = v1pb.DataSource_CLUSTER
	default:
	}
	return authenticationType
}

func convertV1DataSource(dataSource *v1pb.DataSource) (*storepb.DataSource, error) {
	dsType, err := convertV1DataSourceType(dataSource.Type)
	if err != nil {
		return nil, err
	}
	externalSecret, err := convertV1DataSourceExternalSecret(dataSource.ExternalSecret)
	if err != nil {
		return nil, err
	}
	saslConfig := convertV1DataSourceSaslConfig(dataSource.SaslConfig)

	storeDataSource := &storepb.DataSource{
		Id:                                 dataSource.Id,
		Type:                               dsType,
		Username:                           dataSource.Username,
		Password:                           dataSource.Password,
		SslCa:                              dataSource.SslCa,
		SslCert:                            dataSource.SslCert,
		SslKey:                             dataSource.SslKey,
		Host:                               dataSource.Host,
		Port:                               dataSource.Port,
		Database:                           dataSource.Database,
		Srv:                                dataSource.Srv,
		AuthenticationDatabase:             dataSource.AuthenticationDatabase,
		Sid:                                dataSource.Sid,
		ServiceName:                        dataSource.ServiceName,
		SshHost:                            dataSource.SshHost,
		SshPort:                            dataSource.SshPort,
		SshUser:                            dataSource.SshUser,
		SshPassword:                        dataSource.SshPassword,
		SshPrivateKey:                      dataSource.SshPrivateKey,
		AuthenticationPrivateKey:           dataSource.AuthenticationPrivateKey,
		AuthenticationPrivateKeyPassphrase: dataSource.AuthenticationPrivateKeyPassphrase,
		ExternalSecret:                     externalSecret,
		SaslConfig:                         saslConfig,
		AuthenticationType:                 convertV1AuthenticationType(dataSource.AuthenticationType),
		AdditionalAddresses:                convertAdditionalAddresses(dataSource.AdditionalAddresses),
		ReplicaSet:                         dataSource.ReplicaSet,
		DirectConnection:                   dataSource.DirectConnection,
		Region:                             dataSource.Region,
		WarehouseId:                        dataSource.WarehouseId,
		UseSsl:                             dataSource.UseSsl,
		VerifyTlsCertificate:               dataSource.VerifyTlsCertificate,
		RedisType:                          convertV1RedisType(dataSource.RedisType),
		MasterName:                         dataSource.MasterName,
		MasterUsername:                     dataSource.MasterUsername,
		MasterPassword:                     dataSource.MasterPassword,
		ExtraConnectionParameters:          dataSource.ExtraConnectionParameters,
	}

	switch dataSource.AuthenticationType {
	case v1pb.DataSource_AZURE_IAM:
		if azureCredential := dataSource.GetAzureCredential(); azureCredential != nil {
			storeDataSource.IamExtension = &storepb.DataSource_AzureCredential_{
				AzureCredential: &storepb.DataSource_AzureCredential{
					TenantId:     azureCredential.TenantId,
					ClientId:     azureCredential.ClientId,
					ClientSecret: azureCredential.ClientSecret,
				},
			}
		}
	case v1pb.DataSource_AWS_RDS_IAM:
		if awsCredential := dataSource.GetAwsCredential(); awsCredential != nil {
			storeDataSource.IamExtension = &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{
					AccessKeyId:     awsCredential.AccessKeyId,
					SecretAccessKey: awsCredential.SecretAccessKey,
					SessionToken:    awsCredential.SessionToken,
					RoleArn:         awsCredential.RoleArn,
					ExternalId:      awsCredential.ExternalId,
				},
			}
		}
	case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		if gcpCredential := dataSource.GetGcpCredential(); gcpCredential != nil {
			storeDataSource.IamExtension = &storepb.DataSource_GcpCredential{
				GcpCredential: &storepb.DataSource_GCPCredential{
					Content: gcpCredential.Content,
				},
			}
		}
	default:
	}

	return storeDataSource, nil
}

func convertV1DataSourceType(tp v1pb.DataSourceType) (storepb.DataSourceType, error) {
	switch tp {
	case v1pb.DataSourceType_READ_ONLY:
		return storepb.DataSourceType_READ_ONLY, nil
	case v1pb.DataSourceType_ADMIN:
		return storepb.DataSourceType_ADMIN, nil
	default:
		return storepb.DataSourceType_DATA_SOURCE_UNSPECIFIED, errors.Errorf("invalid data source type %v", tp)
	}
}
