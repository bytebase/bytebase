import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
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
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type {
  DataSource,
  Instance,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSourceType,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { convertKVListToLabels, convertLabelsToKVList } from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/connect";
import { FeatureModal } from "../ui/feature-modal";
import {
  type ConnectionFailureCategory,
  connectionFailureCategoryHeader,
  normalizeConnectionFailureCategory,
} from "./ConnectionRecovery";
import type { BasicInfo, DataSourceEditState, EditDataSource } from "./common";
import {
  calcDataSourceUpdateMask,
  extractBasicInfo,
  extractDataSourceEditState,
  krbConfigOf,
  movesKeytabToNewDestination,
} from "./common";
import { effectivePortForEngine } from "./constants";
import { hasInstancePermission } from "./permission";
import { type InstanceSpecs, useInstanceSpecs } from "./specs";
import { type ValidationErrors, validateDataSource } from "./validation";

export type LocalState = {
  editingDataSourceId: string | undefined;
  isTestingConnection: boolean;
  isRequesting: boolean;
};

const localConnectionHosts = new Set([
  "localhost",
  "0.0.0.0",
  "::1",
  "[::1]",
  "host.docker.internal",
]);

const isLocalConnectionHost = (host: string) =>
  localConnectionHosts.has(host) || host.startsWith("127.");

type TestConnectionResult = {
  success: boolean;
  message: string;
  failureCategory: ConnectionFailureCategory;
};

export interface InstanceFormContextValue {
  instance: Instance | undefined;
  parent: string | undefined;
  project: Project | undefined;
  hideAdvancedFeatures: boolean;
  state: LocalState;
  setState: React.Dispatch<React.SetStateAction<LocalState>>;
  specs: InstanceSpecs;
  isCreating: boolean;
  allowEdit: boolean;
  allowCreate: boolean;
  hasPermission: (permission: Permission) => boolean;
  environment: ReturnType<
    ReturnType<typeof useAppStore.getState>["getEnvironmentByName"]
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
  getDataSourceErrors: (dataSource: EditDataSource) => ValidationErrors;
  checkDataSource: (dataSources: EditDataSource[]) => boolean;
  needsKeytabResupply: (dataSource: EditDataSource) => boolean;
  resetDataSource: () => void;
  extractDataSourceFromEdit: (
    engine: Engine,
    edit: EditDataSource
  ) => DataSource;
  testConnection: (
    editingDS: EditDataSource,
    silent?: boolean
  ) => Promise<TestConnectionResult>;
  pendingCreateInstance: Instance;
  valueChanged: boolean;
  isEditing: boolean;
  onDismiss?: () => void;
  dataSourceResetEvent: number;
  emitDataSourceReset: () => void;
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
  parent,
  project,
  hideAdvancedFeatures = false,
  onDismiss,
  children,
}: {
  instance?: Instance;
  parent?: string;
  project?: Project;
  hideAdvancedFeatures?: boolean;
  onDismiss?: () => void;
  children: ReactNode;
}) {
  const { t } = useTranslation();
  const environmentList = useAppStore((s) => s.environmentList);
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());

  const [state, setState] = useState<LocalState>(() => ({
    editingDataSourceId: instance?.dataSources.find(
      (ds) => ds.type === DataSourceType.ADMIN
    )?.id,
    isTestingConnection: false,
    isRequesting: false,
  }));

  const [basicInfo, setBasicInfo] = useState<BasicInfo>(() => {
    const info = extractBasicInfo(instance);
    if (!instance && parent) {
      info.name = `${parent}/${info.name}`;
    }
    return info;
  });
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
  const [dataSourceResetEvent, setDataSourceResetEvent] = useState(0);
  const syncedInstanceRef = useRef({
    name: instance?.name,
    state: instance?.state,
  });
  const isCreating = instance === undefined;

  useEffect(() => {
    const previous = syncedInstanceRef.current;
    const next = {
      name: instance?.name,
      state: instance?.state,
    };
    if (previous.name === next.name && previous.state === next.state) {
      return;
    }

    syncedInstanceRef.current = next;
    const nextBasicInfo = extractBasicInfo(instance);
    const nextDataSourceEditState = extractDataSourceEditState(instance);
    setBasicInfo(nextBasicInfo);
    setLabelKVList(convertLabelsToKVList(nextBasicInfo.labels, true));
    setDataSourceEditState(nextDataSourceEditState);
    setState((prev) => ({
      ...prev,
      editingDataSourceId: nextDataSourceEditState.editingDataSourceId,
    }));
    setLabelErrors([]);
    setMissingFeature(undefined);
  }, [instance]);

  useEffect(() => {
    if (!isCreating || basicInfo.environment) {
      return;
    }

    const firstEnvironment = environmentList[0];
    if (!firstEnvironment) {
      return;
    }

    setBasicInfo((prev) =>
      prev.environment ? prev : { ...prev, environment: firstEnvironment.name }
    );
  }, [basicInfo.environment, environmentList, isCreating]);

  const emitShowConnectionOptions = useCallback(() => {
    setShowConnectionOptionsEvent((prev) => prev + 1);
  }, []);

  const hasPermission = useCallback(
    (permission: Permission) => hasInstancePermission(project, permission),
    [project]
  );

  const allowEdit = isCreating
    ? true
    : (instance?.state || State.STATE_UNSPECIFIED) === State.ACTIVE &&
      hasPermission("bb.instances.update");

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

  const hasReadonlyReplicaFeature = useAppStore((s) =>
    s.hasInstanceFeature(
      PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION,
      instance
    )
  );

  const specs = useInstanceSpecs(basicInfo, adminDataSource, editingDataSource);

  const environment = useMemo(
    () =>
      useAppStore.getState().getEnvironmentByName(basicInfo.environment ?? ""),
    // Re-resolve when the environment cache loads (environmentList) or the
    // selected environment changes.
    [basicInfo.environment, environmentList]
  );

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
          "updatedToken",
          "updateSsl"
        )
      );
      if (edit.updatedPassword) ds.password = edit.updatedPassword;
      if (edit.useEmptyPassword) ds.password = "";
      if (edit.updatedMasterPassword)
        ds.masterPassword = edit.updatedMasterPassword;
      if (edit.useEmptyMasterPassword) ds.masterPassword = "";
      if (edit.updatedToken) ds.authenticationPrivateKey = edit.updatedToken;
      ds.port = effectivePortForEngine(engine, ds.port, ds.srv);
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

  // Asks movesKeytabToNewDestination about one data source of the form,
  // resolving both sides it compares: the stored data source by ID, and the
  // value this edit would send. The Kerberos filter comes first so that an
  // ordinary data source never pays for that extraction.
  const needsKeytabResupply = useCallback(
    (ds: EditDataSource): boolean => {
      if (!krbConfigOf(ds)) return false;
      return movesKeytabToNewDestination(
        extractDataSourceFromEdit(basicInfo.engine, ds),
        instance?.dataSources.find((d) => d.id === ds.id)
      );
    },
    [basicInfo.engine, extractDataSourceFromEdit, instance]
  );

  const getDataSourceErrors = useCallback(
    (ds: EditDataSource) =>
      validateDataSource(ds, {
        engine: basicInfo.engine,
        isSaaSMode,
        stored: instance?.dataSources.find((stored) => stored.id === ds.id),
        keytabResupply: needsKeytabResupply(ds),
      }),
    [basicInfo.engine, isSaaSMode, instance, needsKeytabResupply]
  );

  const checkDataSource = useCallback(
    (dataSources: EditDataSource[]) =>
      dataSources.every(
        (ds) => Object.keys(getDataSourceErrors(ds)).length === 0
      ),
    [getDataSourceErrors]
  );

  const allowCreate =
    hasPermission("bb.instances.create") &&
    !!basicInfo.title.trim() &&
    resourceIdValidated &&
    labelErrors.length === 0 &&
    checkDataSource([adminDataSource]);

  const emitDataSourceReset = useCallback(() => {
    setDataSourceResetEvent((event) => event + 1);
  }, []);

  const resetDataSource = useCallback(() => {
    setDataSourceEditState(extractDataSourceEditState(instance));
    emitDataSourceReset();
  }, [instance, emitDataSourceReset]);

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
    ): Promise<TestConnectionResult> => {
      const ok = (): TestConnectionResult => {
        if (!silent) {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("instance.successfully-connected-instance"),
          });
        }
        setState((prev) => ({ ...prev, isTestingConnection: false }));
        return { success: true, message: "", failureCategory: "unknown" };
      };
      const fail = (host: string, err: unknown): TestConnectionResult => {
        let failureCategory = normalizeConnectionFailureCategory(
          err instanceof ConnectError
            ? err.metadata.get(connectionFailureCategoryHeader)
            : undefined
        );
        // Gateways may return a bare HTTP error without Bytebase's category.
        // Connect maps 504 to Unavailable, which also covers non-timeout errors.
        if (
          failureCategory === "unknown" &&
          err instanceof ConnectError &&
          (err.code === Code.DeadlineExceeded || err.rawMessage === "HTTP 504")
        ) {
          failureCategory = "timeout";
        }
        let error =
          err instanceof ConnectError
            ? err.rawMessage
            : extractGrpcErrorMessage(err);
        if (!silent) {
          const normalizedHost = host.trim().toLowerCase();
          if (isSaaSMode && isLocalConnectionHost(normalizedHost)) {
            error = `${error}\n\n${t(
              "instance.failed-to-connect-instance-saas-local-host",
              { host }
            )}`;
          } else if (
            isLocalConnectionHost(normalizedHost) &&
            normalizedHost !== "host.docker.internal"
          ) {
            error = `${error}\n\n${t("instance.failed-to-connect-instance-localhost")}`;
          }
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title:
              failureCategory === "timeout"
                ? t("instance.connection-recovery.timeout.test-title")
                : t("instance.failed-to-connect-instance"),
            description:
              failureCategory === "timeout"
                ? `${t(
                    isSaaSMode
                      ? "instance.connection-recovery.timeout.description-saas"
                      : "instance.connection-recovery.timeout.description-self-hosted"
                  )}\n\n${t("error-page.error-details")}: ${error}`
                : error,
            manualHide: true,
          });
        }
        setState((prev) => ({ ...prev, isTestingConnection: false }));
        return { success: false, message: error, failureCategory };
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
          await useAppStore
            .getState()
            .createInstance(inst, true, parent ? { parent } : undefined);
          return ok();
        } catch (err) {
          return fail(dataSourceCreate.host, err);
        }
      } else {
        const ds = extractDataSourceFromEdit(instance!.engine, editingDS);
        if (editingDS.pendingCreate) {
          try {
            await useAppStore.getState().createDataSource({
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
            await useAppStore.getState().updateDataSource({
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
      extractDataSourceFromEdit,
      parent,
      isSaaSMode,
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
      parent,
      project,
      hideAdvancedFeatures,
      state,
      setState,
      specs,
      isCreating,
      allowEdit,
      allowCreate,
      hasPermission,
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
      getDataSourceErrors,
      needsKeytabResupply,
      resetDataSource,
      extractDataSourceFromEdit,
      testConnection,
      pendingCreateInstance,
      valueChanged,
      isEditing,
      onDismiss,
      dataSourceResetEvent,
      emitDataSourceReset,
      showConnectionOptionsEvent,
      emitShowConnectionOptions,
    }),
    [
      instance,
      parent,
      project,
      hideAdvancedFeatures,
      state,
      specs,
      isCreating,
      allowEdit,
      allowCreate,
      hasPermission,
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
      getDataSourceErrors,
      needsKeytabResupply,
      resetDataSource,
      extractDataSourceFromEdit,
      testConnection,
      pendingCreateInstance,
      valueChanged,
      isEditing,
      onDismiss,
      dataSourceResetEvent,
      emitDataSourceReset,
      showConnectionOptionsEvent,
      emitShowConnectionOptions,
    ]
  );

  return (
    <InstanceFormCtx.Provider value={value}>
      {children}
      <FeatureModal
        open={!!missingFeature}
        feature={missingFeature}
        instance={instance}
        onOpenChange={(open) => {
          if (!open) setMissingFeature(undefined);
        }}
      />
    </InstanceFormCtx.Provider>
  );
}
