import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual, omit } from "lodash-es";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
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
import type { BasicInfo, DataSourceEditState, EditDataSource } from "./common";
import {
  calcDataSourceUpdateMask,
  extractBasicInfo,
  extractDataSourceEditState,
} from "./common";
import { type InstanceSpecs, useInstanceSpecs } from "./specs";

export type LocalState = {
  editingDataSourceId: string | undefined;
  isTestingConnection: boolean;
  isRequesting: boolean;
};

export interface InstanceFormContextValue {
  instance: Instance | undefined;
  hideAdvancedFeatures: boolean;
  state: LocalState;
  setState: React.Dispatch<React.SetStateAction<LocalState>>;
  specs: InstanceSpecs;
  isCreating: boolean;
  allowEdit: boolean;
  allowCreate: boolean;
  environment: ReturnType<
    ReturnType<typeof useEnvironmentV1Store>["getEnvironmentByName"]
  >;
  basicInfo: BasicInfo;
  setBasicInfo: React.Dispatch<React.SetStateAction<BasicInfo>>;
  labelKVList: { key: string; value: string }[];
  setLabelKVList: React.Dispatch<
    React.SetStateAction<{ key: string; value: string }[]>
  >;
  dataSourceEditState: DataSourceEditState;
  setDataSourceEditState: React.Dispatch<
    React.SetStateAction<DataSourceEditState>
  >;
  adminDataSource: EditDataSource;
  editingDataSource: EditDataSource | undefined;
  readonlyDataSourceList: EditDataSource[];
  hasReadOnlyDataSource: boolean;
  hasReadonlyReplicaFeature: boolean;
  missingFeature: PlanFeature | undefined;
  setMissingFeature: React.Dispatch<
    React.SetStateAction<PlanFeature | undefined>
  >;
  resourceIdValidated: boolean;
  setResourceIdValidated: React.Dispatch<React.SetStateAction<boolean>>;
  labelErrors: string[];
  setLabelErrors: React.Dispatch<React.SetStateAction<string[]>>;
  checkDataSource: (dataSources: DataSource[]) => boolean;
  resetDataSource: () => void;
  extractDataSourceFromEdit: (
    engine: Engine,
    edit: EditDataSource
  ) => DataSource;
  testConnection: (
    editingDS: EditDataSource,
    silent?: boolean
  ) => Promise<{ success: boolean; message: string }>;
  pendingCreateInstance: Instance;
  valueChanged: boolean;
  isEditing: boolean;
  onDismiss?: () => void;
  showConnectionOptionsEvent: number;
  emitShowConnectionOptions: () => void;
}

const InstanceFormCtx = createContext<InstanceFormContextValue | null>(null);

export const useInstanceFormContext = () => {
  const ctx = useContext(InstanceFormCtx);
  if (!ctx) {
    throw new Error(
      "useInstanceFormContext must be used within InstanceFormProvider"
    );
  }
  return ctx;
};

export function InstanceFormProvider({
  instance,
  hideAdvancedFeatures = false,
  onDismiss,
  children,
}: {
  instance?: Instance;
  hideAdvancedFeatures?: boolean;
  onDismiss?: () => void;
  children: ReactNode;
}) {
  const { t } = useTranslation();
  const instanceStore = useInstanceV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const environmentStore = useEnvironmentV1Store();

  const [state, setState] = useState<LocalState>(() => ({
    editingDataSourceId: instance?.dataSources.find(
      (ds) => ds.type === DataSourceType.ADMIN
    )?.id,
    isTestingConnection: false,
    isRequesting: false,
  }));

  const [basicInfo, setBasicInfo] = useState<BasicInfo>(() =>
    extractBasicInfo(instance)
  );
  const [dataSourceEditState, setDataSourceEditState] =
    useState<DataSourceEditState>(() => extractDataSourceEditState(instance));
  const [labelKVList, setLabelKVList] = useState(() =>
    convertLabelsToKVList(basicInfo.labels, true)
  );
  const [missingFeature, setMissingFeature] = useState<
    PlanFeature | undefined
  >();
  const [resourceIdValidated, setResourceIdValidated] = useState(false);
  const [labelErrors, setLabelErrors] = useState<string[]>([]);
  const [showConnectionOptionsEvent, setShowConnectionOptionsEvent] =
    useState(0);

  const emitShowConnectionOptions = useCallback(() => {
    setShowConnectionOptionsEvent((prev) => prev + 1);
  }, []);

  const isCreating = instance === undefined;
  const allowEdit = isCreating
    ? true
    : (instance?.state || State.STATE_UNSPECIFIED) === State.ACTIVE &&
      hasWorkspacePermissionV2("bb.instances.update");

  const adminDataSource = useMemo(
    () =>
      dataSourceEditState.dataSources.find(
        (ds) => ds.type === DataSourceType.ADMIN
      )!,
    [dataSourceEditState.dataSources]
  );

  const editingDataSource = useMemo(() => {
    const { dataSources, editingDataSourceId } = dataSourceEditState;
    if (editingDataSourceId === undefined) return undefined;
    return dataSources.find((ds) => ds.id === editingDataSourceId);
  }, [dataSourceEditState]);

  const readonlyDataSourceList = useMemo(
    () =>
      dataSourceEditState.dataSources.filter(
        (ds) => ds.type === DataSourceType.READ_ONLY
      ),
    [dataSourceEditState.dataSources]
  );

  const hasReadOnlyDataSource = readonlyDataSourceList.length > 0;

  const hasReadonlyReplicaFeature = useMemo(
    () =>
      subscriptionStore.hasInstanceFeature(
        PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION,
        instance
      ),
    [subscriptionStore, instance]
  );

  const specs = useInstanceSpecs(basicInfo, adminDataSource, editingDataSource);

  const environment = useMemo(
    () => environmentStore.getEnvironmentByName(basicInfo.environment ?? ""),
    [environmentStore, basicInfo.environment]
  );

  const checkDataSource = useCallback(
    (dataSources: DataSource[]) => {
      return dataSources.every((ds) => {
        if (
          ds.authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return /.+:.+:.+/.test(ds.host);
        }
        if (
          ds.authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return !!ds.region;
        }
        if (basicInfo.engine === Engine.ORACLE) {
          if (!ds.sid && !ds.serviceName) return false;
        } else if (basicInfo.engine === Engine.DATABRICKS) {
          if (!ds.warehouseId || !ds.authenticationPrivateKey) return false;
        }
        if (ds.saslConfig?.mechanism?.case === "krbConfig") {
          const krbConfig = ds.saslConfig.mechanism.value;
          if (
            !krbConfig.primary ||
            !krbConfig.realm ||
            !krbConfig.kdcHost ||
            !krbConfig.keytab
          )
            return false;
        }
        if (!ds.externalSecret) return true;
        switch (ds.externalSecret.secretType) {
          case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
            if (
              !ds.externalSecret.url ||
              !ds.externalSecret.engineName ||
              !ds.externalSecret.secretName ||
              !ds.externalSecret.passwordKeyName
            )
              return false;
            break;
          case DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER:
            if (
              !ds.externalSecret.secretName ||
              !ds.externalSecret.passwordKeyName
            )
              return false;
            break;
          case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
            if (!ds.externalSecret.secretName) return false;
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
    },
    [basicInfo.engine]
  );

  const allowCreate = useMemo(() => {
    if (!hasWorkspacePermissionV2("bb.instances.create")) return false;
    if (basicInfo.engine === Engine.SPANNER) {
      return (
        !!basicInfo.title.trim() && isValidSpannerHost(adminDataSource.host)
      );
    }
    if (basicInfo.engine === Engine.BIGQUERY) {
      return !!basicInfo.title.trim() && adminDataSource.host !== "";
    }
    if (basicInfo.engine !== Engine.DYNAMODB) {
      if (adminDataSource.host === "") return false;
    }
    if (basicInfo.engine === Engine.REDIS) {
      if (
        adminDataSource.redisType === DataSource_RedisType.SENTINEL &&
        adminDataSource.masterName === ""
      )
        return false;
    }
    const hasLabelErrs = labelErrors.length > 0;
    return (
      !!basicInfo.title.trim() &&
      resourceIdValidated &&
      checkDataSource([adminDataSource]) &&
      !hasLabelErrs
    );
  }, [
    basicInfo,
    adminDataSource,
    resourceIdValidated,
    labelErrors,
    checkDataSource,
  ]);

  const resetDataSource = useCallback(() => {
    setDataSourceEditState(extractDataSourceEditState(instance));
  }, [instance]);

  const extractDataSourceFromEdit = useCallback(
    (engine: Engine, edit: EditDataSource): DataSource => {
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
      if (edit.updatedPassword) ds.password = edit.updatedPassword;
      if (edit.useEmptyPassword) ds.password = "";
      if (edit.updatedMasterPassword)
        ds.masterPassword = edit.updatedMasterPassword;
      if (edit.useEmptyMasterPassword) ds.masterPassword = "";
      if (!specs.showDatabase) ds.database = "";
      if (engine !== Engine.ORACLE) {
        ds.sid = "";
        ds.serviceName = "";
      }
      if (engine !== Engine.MONGODB) {
        ds.srv = false;
        ds.authenticationDatabase = "";
      }
      if (!specs.showSSH) {
        ds.sshHost = "";
        ds.sshPort = "";
        ds.sshUser = "";
        ds.sshPassword = "";
        ds.sshPrivateKey = "";
      }
      if (!specs.showSSL) {
        ds.sslCa = "";
        ds.sslCert = "";
        ds.sslKey = "";
      }
      return ds;
    },
    [specs]
  );

  // Debounced to avoid expensive cloneDeep + extraction on every keystroke.
  const [pendingCreateInstance, setPendingCreateInstance] = useState<Instance>(
    () => create(InstanceSchema, {})
  );
  const pendingTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => {
    clearTimeout(pendingTimerRef.current);
    pendingTimerRef.current = setTimeout(() => {
      const currentLabels = convertKVListToLabels(labelKVList, false);
      const inst: Instance = create(InstanceSchema, {
        ...basicInfo,
        labels: currentLabels,
        engineVersion: "",
        dataSources: [],
      });
      if (editingDataSource) {
        const dataSourceCreate = extractDataSourceFromEdit(
          inst.engine,
          adminDataSource
        );
        inst.dataSources = [dataSourceCreate];
      }
      setPendingCreateInstance(inst);
    }, 300);
    return () => clearTimeout(pendingTimerRef.current);
  }, [
    basicInfo,
    labelKVList,
    editingDataSource,
    adminDataSource,
    extractDataSourceFromEdit,
  ]);

  // Use ref for testConnection to avoid stale closures
  const stateRef = useRef(state);
  stateRef.current = state;

  const testConnection = useCallback(
    async (
      editingDS: EditDataSource,
      silent = false
    ): Promise<{ success: boolean; message: string }> => {
      const ok = () => {
        if (!silent) {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("instance.successfully-connected-instance"),
          });
        }
        setState((prev) => ({ ...prev, isTestingConnection: false }));
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
            manualHide: true,
          });
        }
        setState((prev) => ({ ...prev, isTestingConnection: false }));
        return { success: false, message: error };
      };

      setState((prev) => ({ ...prev, isTestingConnection: true }));

      if (isCreating) {
        const inst: Instance = create(InstanceSchema, {
          ...basicInfo,
          engineVersion: "",
          dataSources: [],
        });
        const dataSourceCreate = extractDataSourceFromEdit(
          inst.engine,
          editingDS
        );
        inst.dataSources = [dataSourceCreate];
        try {
          await instanceStore.createInstance(inst, true);
          return ok();
        } catch (err) {
          return fail(dataSourceCreate.host, err);
        }
      } else {
        const ds = extractDataSourceFromEdit(instance!.engine, editingDS);
        if (editingDS.pendingCreate) {
          try {
            await instanceStore.createDataSource({
              instance: instance!.name,
              dataSource: ds,
              validateOnly: true,
            });
            return ok();
          } catch (err) {
            return fail(ds.host, err);
          }
        } else {
          try {
            const original = instance!.dataSources.find(
              (d) => d.id === editingDS.id
            );
            if (!original) throw new Error("should never reach this line");
            const updateMask = calcDataSourceUpdateMask(
              ds,
              original,
              editingDS
            );
            await instanceStore.updateDataSource({
              instance: instance!.name,
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
    },
    [
      isCreating,
      basicInfo,
      instance,
      instanceStore,
      extractDataSourceFromEdit,
      t,
    ]
  );

  // Debounced valueChanged to avoid expensive deep comparison on every keystroke.
  const [valueChanged, setValueChanged] = useState(false);
  const valueChangedTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => {
    clearTimeout(valueChangedTimerRef.current);
    valueChangedTimerRef.current = setTimeout(() => {
      if (instance?.state === State.DELETED) {
        setValueChanged(false);
        return;
      }
      const original = {
        basicInfo: extractBasicInfo(instance),
        dataSources: extractDataSourceEditState(instance).dataSources,
      };
      const currentLabels = convertKVListToLabels(labelKVList, false);
      const editing = {
        basicInfo: { ...basicInfo, labels: currentLabels },
        dataSources: dataSourceEditState.dataSources,
      };
      setValueChanged(!isEqual(editing, original));
    }, 300);
    return () => clearTimeout(valueChangedTimerRef.current);
  }, [instance, basicInfo, labelKVList, dataSourceEditState.dataSources]);

  const isEditing = valueChanged && allowEdit;

  const value: InstanceFormContextValue = useMemo(
    () => ({
      instance,
      hideAdvancedFeatures,
      state,
      setState,
      specs,
      isCreating,
      allowEdit,
      allowCreate,
      environment,
      basicInfo,
      setBasicInfo,
      labelKVList,
      setLabelKVList,
      dataSourceEditState,
      setDataSourceEditState,
      adminDataSource,
      editingDataSource,
      readonlyDataSourceList,
      hasReadOnlyDataSource,
      hasReadonlyReplicaFeature,
      missingFeature,
      setMissingFeature,
      resourceIdValidated,
      setResourceIdValidated,
      labelErrors,
      setLabelErrors,
      checkDataSource,
      resetDataSource,
      extractDataSourceFromEdit,
      testConnection,
      pendingCreateInstance,
      valueChanged,
      isEditing,
      onDismiss,
      showConnectionOptionsEvent,
      emitShowConnectionOptions,
    }),
    [
      instance,
      hideAdvancedFeatures,
      state,
      specs,
      isCreating,
      allowEdit,
      allowCreate,
      environment,
      basicInfo,
      labelKVList,
      dataSourceEditState,
      adminDataSource,
      editingDataSource,
      readonlyDataSourceList,
      hasReadOnlyDataSource,
      hasReadonlyReplicaFeature,
      missingFeature,
      resourceIdValidated,
      labelErrors,
      checkDataSource,
      resetDataSource,
      extractDataSourceFromEdit,
      testConnection,
      pendingCreateInstance,
      valueChanged,
      isEditing,
      onDismiss,
      showConnectionOptionsEvent,
      emitShowConnectionOptions,
    ]
  );

  return (
    <InstanceFormCtx.Provider value={value}>
      {children}
    </InstanceFormCtx.Provider>
  );
}
