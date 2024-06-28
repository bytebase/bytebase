import Emittery from "emittery";
import { cloneDeep, omit } from "lodash-es";
import { useDialog } from "naive-ui";
import type { InjectionKey, Ref } from "vue";
import { provide, inject, computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { instanceServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useCurrentUserV1,
  useEnvironmentV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { UNKNOWN_ID, unknownEnvironment, type FeatureType } from "@/types";
import { Engine, State } from "@/types/proto/v1/common";
import type { DataSource, Instance } from "@/types/proto/v1/instance_service";
import {
  DataSourceExternalSecret_AuthType,
  DataSourceExternalSecret_SecretType,
  DataSourceType,
  DataSource_AuthenticationType,
} from "@/types/proto/v1/instance_service";
import {
  extractInstanceResourceName,
  hasWorkspacePermissionV2,
  isValidSpannerHost,
} from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import type { ResourceIdField } from "../v2";
import type { EditDataSource } from "./common";
import {
  calcDataSourceUpdateMask,
  extractBasicInfo,
  extractDataSourceEditState,
} from "./common";
import { useInstanceSpecs } from "./specs";

export type LocalState = {
  editingDataSourceId: string | undefined;
  isTestingConnection: boolean;
  isRequesting: boolean;
};

const KEY = Symbol(
  "bb.workspace.instance-form"
) as InjectionKey<InstanceFormContext>;

export const provideInstanceFormContext = (baseContext: {
  instance: Ref<Instance | undefined>;
}) => {
  const $d = useDialog();
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const events = new Emittery<{
    dismiss: undefined;
  }>();
  const { instance } = baseContext;
  const state = ref<LocalState>({
    editingDataSourceId: instance.value?.dataSources.find(
      (ds) => ds.type === DataSourceType.ADMIN
    )?.id,
    isTestingConnection: false,
    isRequesting: false,
  });

  const isCreating = computed(() => instance.value === undefined);
  const allowEdit = computed(() => {
    if (isCreating.value) return true;

    return (
      instance.value?.state === State.ACTIVE &&
      hasWorkspacePermissionV2(me.value, "bb.instances.update")
    );
  });
  const basicInfo = ref(extractBasicInfo(instance.value));
  const dataSourceEditState = ref(extractDataSourceEditState(instance.value));

  const adminDataSource = computed(() => {
    return dataSourceEditState.value.dataSources.find(
      (ds) => ds.type === DataSourceType.ADMIN
    )!;
  });
  const editingDataSource = computed(() => {
    const { dataSources, editingDataSourceId } = dataSourceEditState.value;
    if (editingDataSourceId === undefined) return undefined;
    return dataSources.find((ds) => ds.id === editingDataSourceId);
  });
  const readonlyDataSourceList = computed(() => {
    return dataSourceEditState.value.dataSources.filter(
      (ds) => ds.type === DataSourceType.READ_ONLY
    );
  });
  const hasReadOnlyDataSource = computed(() => {
    return readonlyDataSourceList.value.length > 0;
  });

  const hasReadonlyReplicaFeature = computed(() => {
    return useSubscriptionV1Store().hasInstanceFeature(
      "bb.feature.read-replica-connection",
      instance.value
    );
  });

  const resetDataSource = () => {
    dataSourceEditState.value = extractDataSourceEditState(instance.value);
  };
  const missingFeature = ref<FeatureType | undefined>(undefined);

  const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

  const environment = computed(() => {
    return (
      useEnvironmentV1Store().getEnvironmentByName(
        basicInfo.value.environment
      ) ?? unknownEnvironment()
    );
  });

  const checkDataSource = (dataSources: DataSource[]) => {
    return dataSources.every((ds) => {
      if (
        ds.authenticationType ===
        DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
      ) {
        // CloudSQL instance name should be {project}:{region}:{cloud sql name}
        return /.+:.+:.+/.test(ds.host);
      }
      if (ds.authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM) {
        return !!ds.region;
      }

      if (basicInfo.value.engine === Engine.ORACLE) {
        if (!ds.sid && !ds.serviceName) {
          return false;
        }
      } else if (basicInfo.value.engine === Engine.DATABRICKS) {
        if (!ds.warehouseId) {
          return false;
        }
      }

      if (ds.saslConfig?.krbConfig) {
        if (
          !ds.saslConfig.krbConfig.primary ||
          !ds.saslConfig.krbConfig.realm ||
          !ds.saslConfig.krbConfig.kdcHost ||
          !ds.saslConfig.krbConfig.keytab
        ) {
          return false;
        }
      }

      if (!ds.externalSecret) {
        return true;
      }

      switch (ds.externalSecret.secretType) {
        case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
          if (!ds.externalSecret.url || !ds.externalSecret.engineName) {
            return false;
          }
          if (
            !ds.externalSecret.secretName ||
            !ds.externalSecret.passwordKeyName
          ) {
            return false;
          }
          break;
        case DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER:
          if (
            !ds.externalSecret.secretName ||
            !ds.externalSecret.passwordKeyName
          ) {
            return false;
          }
          break;
        case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
          if (!ds.externalSecret.secretName) {
            return false;
          }
          break;
      }

      switch (ds.externalSecret.authType) {
        case DataSourceExternalSecret_AuthType.TOKEN:
          return !!ds.externalSecret.token;
        case DataSourceExternalSecret_AuthType.VAULT_APP_ROLE:
          return (
            !!ds.externalSecret.appRole?.roleId &&
            !!ds.externalSecret.appRole.secretId
          );
      }

      return true;
    });
  };

  const allowCreate = computed(() => {
    if (!hasWorkspacePermissionV2(me.value, "bb.instances.create")) {
      return false;
    }
    if (environment.value.uid === String(UNKNOWN_ID)) {
      return false;
    }
    if (basicInfo.value.engine === Engine.SPANNER) {
      return (
        basicInfo.value.title.trim() &&
        isValidSpannerHost(adminDataSource.value.host) &&
        adminDataSource.value.updatedPassword
      );
    }

    // Check Host
    if (basicInfo.value.engine !== Engine.DYNAMODB) {
      if (adminDataSource.value.host === "") {
        return false;
      }
    }

    return (
      basicInfo.value.title.trim() &&
      resourceIdField.value?.resourceId &&
      resourceIdField.value?.isValidated &&
      checkDataSource([adminDataSource.value])
    );
  });

  const specs = useInstanceSpecs(basicInfo, adminDataSource, editingDataSource);

  const extractDataSourceFromEdit = (
    instance: Instance,
    edit: EditDataSource
  ): DataSource => {
    const { showDatabase, showSSH, showSSL } = specs;
    const ds = cloneDeep(
      omit(
        edit,
        "pendingCreate",
        "updatedPassword",
        "useEmptyPassword",
        "updateSsl",
        "updateSsh",
        "updateAuthenticationPrivateKey"
      )
    );
    if (edit.updatedPassword) {
      ds.password = edit.updatedPassword;
    }
    if (edit.useEmptyPassword) {
      ds.password = "";
    }

    // Clean up unused fields for certain engine types.
    if (!showDatabase.value) {
      ds.database = "";
    }
    if (instance.engine !== Engine.ORACLE) {
      ds.sid = "";
      ds.serviceName = "";
    }
    if (instance.engine !== Engine.MONGODB) {
      ds.srv = false;
      ds.authenticationDatabase = "";
    }
    if (!showSSH.value) {
      ds.sshHost = "";
      ds.sshPort = "";
      ds.sshUser = "";
      ds.sshPassword = "";
      ds.sshPrivateKey = "";
    }
    if (!showSSL.value) {
      ds.sslCa = "";
      ds.sslCert = "";
      ds.sslKey = "";
    }

    return ds;
  };

  const testConnection = async (
    editingDS: EditDataSource,
    silent = false
  ): Promise<{ success: boolean; message: string }> => {
    if (!editingDataSource.value) {
      throw new Error("should never reach this line");
    }

    // In different scenes, we use different methods to test connection.
    const ok = () => {
      if (!silent) {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("instance.successfully-connected-instance"),
        });
      }
      state.value.isTestingConnection = false;
      return { success: true, message: "" };
    };
    const fail = (host: string, err: unknown) => {
      let error = extractGrpcErrorMessage(err);
      if (!silent) {
        if (host === "localhost" || host === "127.0.0.1") {
          error = `${error}\n\n${t("instance.failed-to-connect-instance-localhost")}`;
        }
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("instance.failed-to-connect-instance"),
          description: error,
          // Manual hide, because user may need time to inspect the error
          manualHide: true,
        });
      }
      state.value.isTestingConnection = false;
      return { success: false, message: error };
    };
    state.value.isTestingConnection = true;
    if (isCreating.value) {
      // When creating new instance, use
      // adminDataSource + CreateInstanceRequest.validateOnly = true
      const instance: Instance = {
        ...basicInfo.value,
        engineVersion: "",
        dataSources: [],
      };
      const dataSourceCreate = extractDataSourceFromEdit(instance, editingDS);
      instance.dataSources = [dataSourceCreate];
      try {
        await instanceServiceClient.createInstance(
          {
            instance,
            instanceId: extractInstanceResourceName(instance.name),
            validateOnly: true,
          },
          {
            silent: true,
          }
        );
        return ok();
      } catch (err) {
        return fail(dataSourceCreate.host, err);
      }
    } else {
      // Editing existed instance.
      const ds = extractDataSourceFromEdit(instance.value!, editingDS);
      if (editingDS.pendingCreate) {
        // When read-only data source is about to be created, use
        // editingDataSource + AddDataSourceRequest.validateOnly = true
        try {
          await instanceServiceClient.addDataSource(
            {
              instance: instance.value!.name,
              dataSource: ds,
              validateOnly: true,
            },
            {
              silent: true,
            }
          );
          return ok();
        } catch (err) {
          return fail(ds.host, err);
        }
      } else {
        // When a data source (admin or read-only) has been edited, use
        // editingDataSource + UpdateDataSourceRequest.validateOnly = true
        try {
          const original = instance.value!.dataSources.find(
            (ds) => ds.id === editingDS.id
          );
          if (!original) {
            throw new Error("should never reach this line");
          }
          const updateMask = calcDataSourceUpdateMask(ds, original, editingDS);
          await instanceServiceClient.updateDataSource(
            {
              instance: instance.value!.name,
              dataSource: ds,
              updateMask,
              validateOnly: true,
            },
            {
              silent: true,
            }
          );
          return ok();
        } catch (err) {
          return fail(ds.host, err);
        }
      }
    }
  };

  const context = {
    ...baseContext,
    $d,
    events,
    state,
    specs,
    isCreating,
    allowEdit,
    allowCreate,
    resourceIdField,
    environment,
    basicInfo,
    dataSourceEditState,
    adminDataSource,
    editingDataSource,
    readonlyDataSourceList,
    hasReadOnlyDataSource,
    hasReadonlyReplicaFeature,
    missingFeature,
    checkDataSource,
    resetDataSource,
    extractDataSourceFromEdit,
    testConnection,
  };
  provide(KEY, context);

  return context;
};

export const useInstanceFormContext = () => {
  return inject(KEY)!;
};

export type InstanceFormContext = ReturnType<typeof provideInstanceFormContext>;
