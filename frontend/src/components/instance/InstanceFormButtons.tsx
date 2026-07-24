import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { router } from "@/app/router";
import {
  INSTANCE_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
} from "@/app/router/handles";
import {
  PRODUCT_INTRO_QUERY_KEY,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
} from "@/lib/productIntro";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  DataSource,
  Instance,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSourceType,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  convertKVListToLabels,
  extractInstanceResourceName,
  isValidBigQueryDataSource,
  isValidSpannerDataSource,
} from "@/utils";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "../ui/alert-dialog";
import { Button } from "../ui/button";
import { StickyActionFooter } from "../ui/sticky-action-footer";
import {
  type ConnectionFailureCategory,
  ConnectionRecovery,
} from "./ConnectionRecovery";
import type { EditDataSource } from "./common";
import {
  calcDataSourceUpdateMask,
  extractBasicInfo,
  extractDataSourceEditState,
  hasSslConfig,
} from "./common";
import { useInstanceFormContext } from "./InstanceFormContext";

interface InstanceFormButtonsProps {
  allowCancel?: boolean;
  onCreated?: (instance: Instance) => void;
  onUpdated?: (instance: Instance) => void;
  className?: string;
}

type ConnectionFailureDialogState = {
  open: boolean;
  message: string;
  failureCategory: ConnectionFailureCategory;
};

export function InstanceFormButtons({
  allowCancel = true,
  onCreated,
  onUpdated,
  className,
}: InstanceFormButtonsProps) {
  const { t } = useTranslation();

  const context = useInstanceFormContext();
  const {
    state,
    setState,
    instance,
    isCreating,
    allowEdit,
    allowCreate,
    basicInfo,
    setBasicInfo,
    labelKVList,
    adminDataSource,
    editingDataSource,
    readonlyDataSourceList,
    setDataSourceEditState,
    hasReadonlyReplicaFeature,
    setMissingFeature,
    testConnection,
    checkDataSource,
    extractDataSourceFromEdit,
    valueChanged,
    onDismiss,
    emitShowConnectionOptions,
  } = context;

  const hasExternalSecretFeature = useAppStore((s) =>
    s.hasFeature(PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER)
  );
  const [connectionFailureDialogState, setConnectionFailureDialogState] =
    useState<ConnectionFailureDialogState>({
      open: false,
      message: "",
      failureCategory: "unknown",
    });
  const [connectionFailureResolver, setConnectionFailureResolver] = useState<
    ((confirmed: boolean) => void) | undefined
  >();

  const checkExternalSecretFeature = (dataSources: DataSource[]) => {
    if (hasExternalSecretFeature) return true;
    return dataSources.every(
      (ds) => !ds.externalSecret && !/^{{.+}}$/.test(ds.password)
    );
  };

  const checkRODataSourceFeature = (inst: Instance) => {
    if (hasReadonlyReplicaFeature) return true;
    if (readonlyDataSourceList.length === 0) return true;

    const checkOne = (ds: EditDataSource) => {
      if (ds.pendingCreate) return false;
      const editing = extractDataSourceFromEdit(inst.engine, ds);
      const original = inst.dataSources.find((d) => d.id === ds.id);
      if (original) {
        const updateMask = calcDataSourceUpdateMask(editing, original, ds);
        if (updateMask.length > 0) return false;
      }
      return true;
    };
    return readonlyDataSourceList.every(checkOne);
  };

  const allowUpdate = useMemo((): boolean => {
    if (!valueChanged) return false;
    if (basicInfo.engine === Engine.SPANNER) {
      if (!isValidSpannerDataSource(adminDataSource)) return false;
      if (readonlyDataSourceList.length > 0) {
        if (readonlyDataSourceList.some((ds) => !isValidSpannerDataSource(ds)))
          return false;
      }
      return !!basicInfo.title.trim();
    }
    if (basicInfo.engine === Engine.BIGQUERY) {
      if (!isValidBigQueryDataSource(adminDataSource)) return false;
      if (readonlyDataSourceList.length > 0) {
        if (readonlyDataSourceList.some((ds) => !isValidBigQueryDataSource(ds)))
          return false;
      }
      return !!basicInfo.title.trim();
    }
    return checkDataSource([adminDataSource, ...readonlyDataSourceList]);
  }, [
    valueChanged,
    basicInfo,
    adminDataSource,
    readonlyDataSourceList,
    checkDataSource,
  ]);

  const hasConfiguredConnectionOptions = (ds: EditDataSource): boolean => {
    const hasExtraParameters =
      Object.keys(ds.extraConnectionParameters ?? {}).length > 0;
    const hasSslConfigValue = hasSslConfig(ds);
    const hasSshConfig = !!(
      ds.sshHost ||
      ds.sshPort ||
      ds.sshUser ||
      ds.sshPassword ||
      ds.sshPrivateKey
    );
    return hasExtraParameters || hasSslConfigValue || hasSshConfig;
  };

  const maybeOpenConnectionOptions = (ds: EditDataSource) => {
    if (!hasConfiguredConnectionOptions(ds)) return;
    emitShowConnectionOptions();
  };

  const closeConnectionFailureDialog = (confirmed: boolean) => {
    setConnectionFailureDialogState({
      open: false,
      message: "",
      failureCategory: "unknown",
    });
    connectionFailureResolver?.(confirmed);
    setConnectionFailureResolver(undefined);
  };

  const confirmContinueWithConnectionFailure = async (
    message: string,
    failureCategory: ConnectionFailureCategory
  ): Promise<boolean> => {
    return await new Promise<boolean>((resolve) => {
      setConnectionFailureDialogState({
        open: true,
        message,
        failureCategory,
      });
      setConnectionFailureResolver(() => resolve);
    });
  };

  const getOriginalEditState = () => ({
    basicInfo: extractBasicInfo(instance),
    dataSources: extractDataSourceEditState(instance).dataSources,
  });

  const resetChanges = () => {
    const original = getOriginalEditState();
    setBasicInfo(cloneDeep(original.basicInfo));
    setDataSourceEditState((prev) => ({
      ...prev,
      dataSources: cloneDeep(original.dataSources),
    }));
  };

  const buildCreateInstance = (): Instance => {
    const currentLabels = convertKVListToLabels(labelKVList, false);
    const inst: Instance = create(InstanceSchema, {
      ...basicInfo,
      labels: currentLabels,
      engineVersion: "",
      dataSources: [],
    });
    if (editingDataSource) {
      inst.dataSources = [
        extractDataSourceFromEdit(inst.engine, adminDataSource),
      ];
    }
    return inst;
  };

  const getRouteProjectContext = () => {
    const projectId = router.currentRoute.value.query.project;
    if (typeof projectId !== "string" || projectId.length === 0) {
      return;
    }
    return {
      id: projectId,
      name: `${projectNamePrefix}${projectId}`,
    };
  };

  const projectContext = getRouteProjectContext();

  const doCreate = async () => {
    if (!isCreating) return;

    const payload = buildCreateInstance();
    if (!checkExternalSecretFeature(payload.dataSources)) {
      setMissingFeature(PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER);
      return;
    }

    setState((prev) => ({ ...prev, isRequesting: true }));
    try {
      const createdInstance = await useAppStore
        .getState()
        .createInstance(payload, false, {
          initialDatabaseProject: projectContext?.name,
        });
      if (onCreated) {
        onCreated(createdInstance);
      } else if (projectContext) {
        router.push({
          name: PROJECT_V1_ROUTE_DATABASES,
          params: { projectId: projectContext.id },
          query: {
            [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
            syncingInstance: extractInstanceResourceName(createdInstance.name),
          },
        });
      } else {
        const instanceId = extractInstanceResourceName(createdInstance.name);
        router.push({
          name: INSTANCE_ROUTE_DETAIL,
          params: { instanceId },
          query: { syncingInstance: instanceId },
          hash: "databases",
        });
      }

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "instance.successfully-created-instance-createdinstance-name",
          {
            0: createdInstance.title,
          }
        ),
        description: projectContext
          ? t("db.syncing-databases-for-instance", {
              0: createdInstance.title,
            })
          : undefined,
      });
    } finally {
      setState((prev) => ({ ...prev, isRequesting: false }));
    }
  };

  const tryCreate = async () => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("instance create clicked", {
        routeId: router.currentRoute.value.name?.toString(),
        resource: projectContext?.name,
      })
    );

    const editingDS = adminDataSource;
    const testResult = await testConnection(editingDS, true);
    if (testResult.success) {
      doCreate();
    } else {
      maybeOpenConnectionOptions(editingDS);
      const confirmed = await confirmContinueWithConnectionFailure(
        testResult.message,
        testResult.failureCategory
      );
      if (confirmed) {
        doCreate();
      }
    }
  };

  const updateEditState = (inst: Instance) => {
    setBasicInfo(extractBasicInfo(inst));
    const updatedEditState = extractDataSourceEditState(inst);
    setDataSourceEditState((prev) => {
      const newDataSources = updatedEditState.dataSources;
      const editingId =
        newDataSources.findIndex((ds) => ds.id === prev.editingDataSourceId) >=
        0
          ? prev.editingDataSourceId
          : updatedEditState.editingDataSourceId;
      return {
        dataSources: newDataSources,
        editingDataSourceId: editingId,
      };
    });
  };

  const doUpdate = async () => {
    const inst = instance;
    if (!inst) return;

    if (!checkRODataSourceFeature(inst)) {
      setMissingFeature(PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION);
      return;
    }

    if (!checkExternalSecretFeature([adminDataSource])) {
      setMissingFeature(PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER);
      return;
    }

    if (
      !checkExternalSecretFeature([adminDataSource, ...readonlyDataSourceList])
    ) {
      setMissingFeature(PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER);
      return;
    }

    const pendingRequestRunners: (() => Promise<unknown>)[] = [];

    const maybeQueueUpdateInstanceBasicInfo = () => {
      const currentLabels = convertKVListToLabels(labelKVList, false);
      const instancePatch = create(InstanceSchema, {
        ...inst,
        ...basicInfo,
        labels: currentLabels,
      });
      const updateMask: string[] = [];
      if (instancePatch.title !== inst.title) updateMask.push("title");
      if (instancePatch.externalLink !== inst.externalLink)
        updateMask.push("external_link");
      if (instancePatch.activation !== inst.activation)
        updateMask.push("activation");
      if (instancePatch.environment !== inst.environment)
        updateMask.push("environment");
      if (
        Number(instancePatch.syncInterval?.seconds || 0n) !==
        Number(inst.syncInterval?.seconds || 0n)
      ) {
        updateMask.push("sync_interval");
      }
      if (!isEqual(instancePatch.syncDatabases, inst.syncDatabases))
        updateMask.push("sync_databases");
      if (!isEqual(instancePatch.labels, inst.labels))
        updateMask.push("labels");
      if (updateMask.length === 0) return;

      pendingRequestRunners.push(() =>
        useAppStore
          .getState()
          .updateInstance(instancePatch, updateMask)
          .then(() => {
            if (updateMask.includes("sync_databases")) {
              return refreshInstanceDatabases(instancePatch.name);
            }
          })
      );
    };

    const refreshInstanceDatabases = async (instanceName: string) => {
      await useAppStore.getState().syncInstance(instanceName, true);
      useAppStore.getState().removeCacheByInstance(instanceName);
    };

    const maybeQueueUpdateDataSource = async (
      editing: DataSource,
      original: DataSource | undefined,
      editState: EditDataSource
    ): Promise<boolean | undefined> => {
      if (!original) return;
      const updateMask = calcDataSourceUpdateMask(editing, original, editState);
      if (updateMask.length === 0) return;

      const testResult = await testConnection(editState, true);
      if (!testResult.success) {
        maybeOpenConnectionOptions(editState);
        const continueAnyway = await confirmContinueWithConnectionFailure(
          testResult.message,
          testResult.failureCategory
        );
        if (!continueAnyway) return true;
      }

      pendingRequestRunners.push(() =>
        useAppStore.getState().updateDataSource({
          instance: inst.name,
          dataSource: editing,
          updateMask,
        })
      );
    };

    const maybeQueueUpdateAdminDataSource = async () => {
      const original = inst.dataSources.find(
        (ds) => ds.type === DataSourceType.ADMIN
      );
      const editing = extractDataSourceFromEdit(inst.engine, adminDataSource);
      return await maybeQueueUpdateDataSource(
        editing,
        original,
        adminDataSource
      );
    };

    const maybeQueueUpsertReadonlyDataSources = async (): Promise<
      boolean | undefined
    > => {
      if (readonlyDataSourceList.length === 0) return false;

      for (let i = 0; i < readonlyDataSourceList.length; i++) {
        const editingDS = readonlyDataSourceList[i];
        const patch = extractDataSourceFromEdit(inst.engine, editingDS);

        if (editingDS.pendingCreate) {
          const testResult = await testConnection(editingDS, true);
          if (!testResult.success) {
            maybeOpenConnectionOptions(editingDS);
            const continueAnyway = await confirmContinueWithConnectionFailure(
              testResult.message,
              testResult.failureCategory
            );
            if (!continueAnyway) return true;
          }
          pendingRequestRunners.push(() =>
            useAppStore.getState().createDataSource({
              instance: inst.name,
              dataSource: patch,
            })
          );
        } else {
          const original = inst.dataSources.find(
            (ds) => ds.id === editingDS.id
          );
          const blocked = await maybeQueueUpdateDataSource(
            patch,
            original,
            editingDS
          );
          if (blocked) return true;
        }
      }
    };

    // Prepare pending request runners
    maybeQueueUpdateInstanceBasicInfo();
    if (await maybeQueueUpdateAdminDataSource()) return;
    if (await maybeQueueUpsertReadonlyDataSources()) return;

    if (pendingRequestRunners.length === 0) return;

    setState((prev) => ({ ...prev, isRequesting: true }));
    try {
      for (let i = 0; i < pendingRequestRunners.length; i++) {
        await pendingRequestRunners[i]();
      }

      const updatedInstance = useAppStore
        .getState()
        .getInstanceByName(inst.name);
      updateEditState(updatedInstance);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.successfully-updated-instance-instance-name", {
          0: updatedInstance.title,
        }),
      });

      if (onUpdated) {
        onUpdated(updatedInstance);
      }
    } finally {
      setState((prev) => ({ ...prev, isRequesting: false }));
    }
  };

  const testConnectionForCurrentEditingDS = async () => {
    if (!editingDataSource) return;
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("instance connection test clicked", {
        routeId: router.currentRoute.value.name?.toString(),
      })
    );

    const testResult = await testConnection(editingDataSource, false);
    if (!testResult.success) {
      maybeOpenConnectionOptions(editingDataSource);
    }
  };

  const cancel = () => {
    onDismiss?.();
  };

  const connectionFailureDialog = (
    <AlertDialog
      open={connectionFailureDialogState.open}
      onOpenChange={(next) => {
        if (!next) {
          closeConnectionFailureDialog(false);
        }
      }}
    >
      <AlertDialogContent className="max-w-2xl">
        <AlertDialogTitle>{t("common.warning")}</AlertDialogTitle>
        <ConnectionRecovery
          category={connectionFailureDialogState.failureCategory}
          className="my-2"
        />
        <AlertDialogDescription className="whitespace-pre-wrap break-all">
          {t("instance.unable-to-connect", {
            0: connectionFailureDialogState.message,
          })}
        </AlertDialogDescription>
        <AlertDialogFooter>
          <Button
            appearance="outline"
            onClick={() => closeConnectionFailureDialog(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button onClick={() => closeConnectionFailureDialog(true)}>
            {t("common.continue-anyway")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );

  if (isCreating) {
    return (
      <>
        {connectionFailureDialog}
        <StickyActionFooter
          className={className}
          left={
            allowCancel ? (
              <Button
                appearance="outline"
                disabled={state.isRequesting || state.isTestingConnection}
                onClick={cancel}
              >
                {t("common.cancel")}
              </Button>
            ) : undefined
          }
          right={
            <Button
              disabled={
                !allowCreate || state.isRequesting || state.isTestingConnection
              }
              onClick={tryCreate}
            >
              {state.isRequesting
                ? projectContext
                  ? t("instance.connecting-database-to-project")
                  : t("common.creating")
                : projectContext
                  ? t("instance.connect-database-to-project")
                  : t("common.create")}
            </Button>
          }
        />
      </>
    );
  }

  if (!instance) return null;
  if (!valueChanged || !allowEdit) return null;

  return (
    <>
      {connectionFailureDialog}
      <StickyActionFooter
        className={cn("mt-4", className)}
        left={
          <Button
            appearance="outline"
            disabled={state.isTestingConnection}
            onClick={resetChanges}
          >
            {t("common.cancel")}
          </Button>
        }
        right={
          <>
            <Button
              appearance="secondary"
              disabled={!allowUpdate || state.isRequesting || !allowEdit}
              onClick={testConnectionForCurrentEditingDS}
            >
              {state.isTestingConnection
                ? t("instance.testing-connection")
                : t("instance.test-connection")}
            </Button>
            <Button
              disabled={
                !allowUpdate || state.isRequesting || state.isTestingConnection
              }
              onClick={doUpdate}
            >
              {state.isRequesting ? t("common.updating") : t("common.update")}
            </Button>
          </>
        }
      />
    </>
  );
}
