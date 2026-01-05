import { create } from "@bufbuild/protobuf";
import Emittery from "emittery";
import { cloneDeep, isEqual, omit } from "lodash-es";
import { useDialog } from "naive-ui";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type {
  DataSource,
  Instance,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceExternalSecret_AuthType,
  DataSourceExternalSecret_SecretType,
  DataSourceType,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  convertKVListToLabels,
  convertLabelsToKVList,
  hasWorkspacePermissionV2,
  isValidSpannerHost,
} from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/connect";
import type { LabelListEditor } from "../Label";
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
  hideAdvancedFeatures: Ref<boolean | undefined>;
}) => {
  const instanceStore = useInstanceV1Store();
  const $d = useDialog();
  const { t } = useI18n();
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
      (instance.value?.state || State.STATE_UNSPECIFIED) === State.ACTIVE &&
      hasWorkspacePermissionV2("bb.instances.update")
    );
  });
  const basicInfo = ref(extractBasicInfo(instance.value));
  const dataSourceEditState = ref(extractDataSourceEditState(instance.value));

  const labelListEditorRef = ref<InstanceType<typeof LabelListEditor>>();
  const labelKVList = ref(
    convertLabelsToKVList(basicInfo.value.labels, true /* sort */)
  );

  watch(
    () => basicInfo.value.labels,
    (newLabels) => {
      labelKVList.value = convertLabelsToKVList(newLabels, true /* sort */);
    }
  );

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
      PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION,
      instance.value
    );
  });

  const resetDataSource = () => {
    dataSourceEditState.value = extractDataSourceEditState(instance.value);
  };
  const missingFeature = ref<PlanFeature | undefined>(undefined);

  const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

  const environment = computed(() => {
    return useEnvironmentV1Store().getEnvironmentByName(
      basicInfo.value.environment ?? ""
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
        if (!ds.warehouseId || !ds.authenticationPrivateKey) {
          return false;
        }
      }

      if (ds.saslConfig?.mechanism?.case === "krbConfig") {
        const krbConfig = ds.saslConfig.mechanism.value;
        if (
          !krbConfig.primary ||
          !krbConfig.realm ||
          !krbConfig.kdcHost ||
          !krbConfig.keytab
        ) {
          return false;
        }
      }

      if (!ds.externalSecret) {
        return true;
      }

      switch (ds.externalSecret.secretType) {
        case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
          if (
            !ds.externalSecret.url ||
            !ds.externalSecret.engineName ||
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
          return !!(
            ds.externalSecret.authOption?.case === "token" &&
            ds.externalSecret.authOption.value
          );
        case DataSourceExternalSecret_AuthType.VAULT_APP_ROLE:
          return !!(
            ds.externalSecret.authOption?.case === "appRole" &&
            ds.externalSecret.authOption.value.roleId &&
            ds.externalSecret.authOption.value.secretId
          );
      }

      return true;
    });
  };

  const allowCreate = computed(() => {
    if (!hasWorkspacePermissionV2("bb.instances.create")) {
      return false;
    }
    if (basicInfo.value.engine === Engine.SPANNER) {
      return (
        !!basicInfo.value.title.trim() &&
        isValidSpannerHost(adminDataSource.value.host)
      );
    }
    if (basicInfo.value.engine === Engine.BIGQUERY) {
      return (
        !!basicInfo.value.title.trim() && adminDataSource.value.host !== ""
      );
    }

    // Check Host
    if (basicInfo.value.engine !== Engine.DYNAMODB) {
      if (adminDataSource.value.host === "") {
        return false;
      }
    }

    // Redis Check Master Name
    if (basicInfo.value.engine === Engine.REDIS) {
      if (
        adminDataSource.value.redisType === DataSource_RedisType.SENTINEL &&
        adminDataSource.value.masterName === ""
      ) {
        return false;
      }
    }

    const hasLabelErrors =
      (labelListEditorRef.value?.flattenErrors ?? []).length > 0;

    return (
      !!basicInfo.value.title.trim() &&
      resourceIdField.value?.isValidated &&
      checkDataSource([adminDataSource.value]) &&
      !hasLabelErrors
    );
  });

  const specs = useInstanceSpecs(basicInfo, adminDataSource, editingDataSource);

  const extractDataSourceFromEdit = (
    engine: Engine,
    edit: EditDataSource
  ): DataSource => {
    const { showDatabase, showSSH, showSSL } = specs;
    const ds = cloneDeep(
      omit(
        edit,
        "pendingCreate",
        "updatedPassword",
        "useEmptyPassword",
        "updatedMasterPassword",
        "useEmptyMasterPassword",
        "updateSsl"
      )
    );
    if (edit.updatedPassword) {
      ds.password = edit.updatedPassword;
    }
    if (edit.useEmptyPassword) {
      ds.password = "";
    }

    if (edit.updatedMasterPassword) {
      ds.masterPassword = edit.updatedMasterPassword;
    }
    if (edit.useEmptyMasterPassword) {
      ds.masterPassword = "";
    }

    // Clean up unused fields for certain engine types.
    if (!showDatabase.value) {
      ds.database = "";
    }
    if (engine !== Engine.ORACLE) {
      ds.sid = "";
      ds.serviceName = "";
    }
    if (engine !== Engine.MONGODB) {
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

  const pendingCreateInstance = computed(() => {
    const currentLabels = convertKVListToLabels(labelKVList.value, false);
    const instance: Instance = create(InstanceSchema, {
      ...basicInfo.value,
      labels: currentLabels,
      engineVersion: "",
      dataSources: [],
    });
    if (editingDataSource.value) {
      const dataSourceCreate = extractDataSourceFromEdit(
        instance.engine,
        adminDataSource.value
      );
      instance.dataSources = [dataSourceCreate];
    }
    return instance;
  });

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
      const instance: Instance = create(InstanceSchema, {
        ...basicInfo.value,
        engineVersion: "",
        dataSources: [],
      });
      const dataSourceCreate = extractDataSourceFromEdit(
        instance.engine,
        editingDS
      );
      instance.dataSources = [dataSourceCreate];
      try {
        await instanceStore.createInstance(instance, true /* validateOnly */);
        return ok();
      } catch (err) {
        return fail(dataSourceCreate.host, err);
      }
    } else {
      // Editing existed instance.
      const ds = extractDataSourceFromEdit(instance.value!.engine, editingDS);
      if (editingDS.pendingCreate) {
        // When read-only data source is about to be created, use
        // editingDataSource + AddDataSourceRequest.validateOnly = true
        try {
          await instanceStore.createDataSource({
            instance: instance.value!.name,
            dataSource: ds,
            validateOnly: true,
          });
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
          await instanceStore.updateDataSource({
            instance: instance.value!.name,
            dataSource: ds,
            updateMask,
            validateOnly: true,
          });
          return ok();
        } catch (err) {
          return fail(ds.host, err);
        }
      }
    }
  };

  const valueChanged = computed(() => {
    if (instance.value?.state === State.DELETED) {
      return false;
    }
    const original = {
      basicInfo: extractBasicInfo(instance.value),
      dataSources: extractDataSourceEditState(instance.value).dataSources,
    };
    const currentLabels = convertKVListToLabels(labelKVList.value, false);
    const editing = {
      basicInfo: {
        ...basicInfo.value,
        labels: currentLabels,
      },
      dataSources: dataSourceEditState.value.dataSources,
    };
    return !isEqual(editing, original);
  });

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
    labelListEditorRef,
    labelKVList,
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
    pendingCreateInstance,
    valueChanged,
  };
  provide(KEY, context);

  return context;
};

export const useInstanceFormContext = () => {
  return inject(KEY)!;
};

export type InstanceFormContext = ReturnType<typeof provideInstanceFormContext>;
