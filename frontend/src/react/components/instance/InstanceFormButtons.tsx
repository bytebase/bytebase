import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  pushNotification,
  useDatabaseV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
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
import { convertKVListToLabels, isValidSpannerHost } from "@/utils";
import { Button } from "../ui/button";
import type { EditDataSource } from "./common";
import {
  calcDataSourceUpdateMask,
  extractBasicInfo,
  extractDataSourceEditState,
} from "./common";
import { useInstanceFormContext } from "./InstanceFormContext";

interface InstanceFormButtonsProps {
  allowCancel?: boolean;
  onCreated?: (instance: Instance) => void;
  onUpdated?: (instance: Instance) => void;
  className?: string;
}

export function InstanceFormButtons({
  allowCancel = true,
  onCreated,
  onUpdated,
  className,
}: InstanceFormButtonsProps) {
  const { t } = useTranslation();
  const instanceV1Store = useInstanceV1Store();
  const databaseStore = useDatabaseV1Store();
  const subscriptionStore = useSubscriptionV1Store();

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
    missingFeature: _missingFeature,
    setMissingFeature,
    testConnection,
    checkDataSource,
    extractDataSourceFromEdit,
    pendingCreateInstance,
    valueChanged,
    onDismiss,
    emitShowConnectionOptions,
  } = context;

  const hasExternalSecretFeature = useMemo(
    () =>
      subscriptionStore.hasFeature(PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER),
    [subscriptionStore]
  );

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
      if (!isValidSpannerHost(adminDataSource.host)) return false;
      if (readonlyDataSourceList.length > 0) {
        if (readonlyDataSourceList.some((ds) => !isValidSpannerHost(ds.host)))
          return false;
      }
      return !!basicInfo.title.trim();
    }
    if (basicInfo.engine === Engine.BIGQUERY) {
      if (!adminDataSource.host) return false;
      if (readonlyDataSourceList.length > 0) {
        if (readonlyDataSourceList.some((ds) => !ds.host)) return false;
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
    const hasSslConfig = !!(ds.useSsl || ds.sslCa || ds.sslCert || ds.sslKey);
    const hasSshConfig = !!(
      ds.sshHost ||
      ds.sshPort ||
      ds.sshUser ||
      ds.sshPassword ||
      ds.sshPrivateKey
    );
    return hasExtraParameters || hasSslConfig || hasSshConfig;
  };

  const maybeOpenConnectionOptions = (ds: EditDataSource) => {
    if (!hasConfiguredConnectionOptions(ds)) return;
    emitShowConnectionOptions();
  };

  const confirmContinueWithConnectionFailure = (message: string): boolean => {
    return window.confirm(
      `${t("common.warning")}\n\n${t("instance.unable-to-connect", { 0: message })}\n\n${t("common.continue-anyway")}?`
    );
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

  const doCreate = async () => {
    if (!isCreating) return;

    if (!checkExternalSecretFeature(pendingCreateInstance.dataSources)) {
      setMissingFeature(PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER);
      return;
    }

    setState((prev) => ({ ...prev, isRequesting: true }));
    try {
      const createdInstance = await instanceV1Store.createInstance(
        pendingCreateInstance
      );
      if (onCreated) {
        onCreated(createdInstance);
      } else {
        router.push(`/${createdInstance.name}`);
        onDismiss?.();
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
      });
    } finally {
      setState((prev) => ({ ...prev, isRequesting: false }));
    }
  };

  const tryCreate = async () => {
    const editingDS = adminDataSource;
    const testResult = await testConnection(editingDS, true);
    if (testResult.success) {
      doCreate();
    } else {
      maybeOpenConnectionOptions(editingDS);
      const confirmed = confirmContinueWithConnectionFailure(
        testResult.message
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
        instanceV1Store.updateInstance(instancePatch, updateMask).then(() => {
          if (updateMask.includes("sync_databases")) {
            return refreshInstanceDatabases(instancePatch.name);
          }
        })
      );
    };

    const refreshInstanceDatabases = async (instanceName: string) => {
      await instanceV1Store.syncInstance(instanceName, true);
      databaseStore.removeCacheByInstance(instanceName);
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
        const continueAnyway = confirmContinueWithConnectionFailure(
          testResult.message
        );
        if (!continueAnyway) return true;
      }

      pendingRequestRunners.push(() =>
        instanceV1Store.updateDataSource({
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
            const continueAnyway = confirmContinueWithConnectionFailure(
              testResult.message
            );
            if (!continueAnyway) return true;
          }
          pendingRequestRunners.push(() =>
            instanceV1Store.createDataSource({
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

      const updatedInstance = instanceV1Store.getInstanceByName(inst.name);
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
    const testResult = await testConnection(editingDataSource, false);
    if (!testResult.success) {
      maybeOpenConnectionOptions(editingDataSource);
    }
  };

  const cancel = () => {
    onDismiss?.();
  };

  if (isCreating) {
    return (
      <div
        className={cn(
          "w-full py-4 border-t border-block-border flex justify-between bg-white",
          className
        )}
      >
        {allowCancel && (
          <Button
            variant="outline"
            disabled={state.isRequesting || state.isTestingConnection}
            onClick={cancel}
          >
            {t("common.cancel")}
          </Button>
        )}
        <div className="flex items-center gap-x-2">
          <Button
            disabled={
              !allowCreate || state.isRequesting || state.isTestingConnection
            }
            onClick={tryCreate}
          >
            {state.isRequesting ? t("common.creating") : t("common.create")}
          </Button>
        </div>
      </div>
    );
  }

  if (!instance) return null;
  if (!valueChanged || !allowEdit) return null;

  return (
    <div
      className={cn(
        "w-full mt-4 py-4 border-t border-block-border flex justify-between bg-white",
        className
      )}
    >
      <Button
        variant="outline"
        disabled={state.isTestingConnection}
        onClick={resetChanges}
      >
        {t("common.cancel")}
      </Button>
      <div className="flex items-center gap-x-2">
        <Button
          variant="ghost"
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
      </div>
    </div>
  );
}
