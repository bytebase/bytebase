<template>
  <template v-if="isCreating">
    <div class="w-full flex justify-between items-center" v-bind="$attrs">
      <div class="w-full flex justify-end items-center gap-x-2">
        <NButton
          v-if="allowCancel"
          quaternary
          :disabled="state.isRequesting || state.isTestingConnection"
          @click.prevent="cancel"
        >
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          :disabled="
            !allowCreate || state.isRequesting || state.isTestingConnection
          "
          :loading="state.isRequesting"
          type="primary"
          @click.prevent="tryCreate"
        >
          {{ $t("common.create") }}
        </NButton>
      </div>
    </div>
  </template>
  <template v-if="!isCreating && instance">
    <div
      v-if="valueChanged && allowEdit"
      class="w-full mt-4 py-4 border-t border-block-border flex justify-between bg-white"
      v-bind="$attrs"
    >
      <NButton
        :disabled="state.isTestingConnection"
        @click.prevent="resetChanges"
      >
        <span> {{ $t("common.cancel") }}</span>
      </NButton>
      <NButton
        :disabled="
          !allowUpdate || state.isRequesting || state.isTestingConnection
        "
        :loading="state.isRequesting"
        type="primary"
        @click.prevent="doUpdate"
      >
        {{ $t("common.confirm-and-update") }}
      </NButton>
    </div>
  </template>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
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
import { convertKVListToLabels, defer, isValidSpannerHost } from "@/utils";
import {
  calcDataSourceUpdateMask,
  type EditDataSource,
  extractBasicInfo,
  extractDataSourceEditState,
} from "./common";
import { useInstanceFormContext } from "./context";
import ScanIntervalInput from "./ScanIntervalInput.vue";

const props = withDefaults(
  defineProps<{
    allowCancel?: boolean;
    onCreated?: (instance: Instance) => void;
    onUpdated?: (instance: Instance) => void;
  }>(),
  {
    allowCancel: true,
    onCreated: undefined,
    onUpdated: undefined,
  }
);

const context = useInstanceFormContext();
const {
  $d,
  events,
  state,
  instance,
  isCreating,
  allowEdit,
  allowCreate,
  basicInfo,
  labelKVList,
  adminDataSource,
  readonlyDataSourceList,
  dataSourceEditState,
  hasReadonlyReplicaFeature,
  missingFeature,
  testConnection,
  checkDataSource,
  extractDataSourceFromEdit,
  pendingCreateInstance,
  valueChanged,
} = context;

const router = useRouter();
const { t } = useI18n();
const instanceV1Store = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const subscriptionStore = useSubscriptionV1Store();
const scanIntervalInputRef = ref<InstanceType<typeof ScanIntervalInput>>();

const resetChanges = () => {
  const original = getOriginalEditState();
  basicInfo.value = cloneDeep(original.basicInfo);
  dataSourceEditState.value.dataSources = cloneDeep(original.dataSources);
};

const allowUpdate = computed((): boolean => {
  if (!valueChanged.value) {
    return false;
  }
  if (scanIntervalInputRef.value) {
    const scanIntervalInput = scanIntervalInputRef.value;
    if (!scanIntervalInput.validate()) {
      return false;
    }
  }
  if (basicInfo.value.engine === Engine.SPANNER) {
    if (!isValidSpannerHost(adminDataSource.value.host)) {
      return false;
    }
    if (readonlyDataSourceList.value.length > 0) {
      if (
        readonlyDataSourceList.value.some((ds) => !isValidSpannerHost(ds.host))
      ) {
        return false;
      }
    }
    return !!basicInfo.value.title.trim();
  }
  if (basicInfo.value.engine === Engine.BIGQUERY) {
    if (!adminDataSource.value.host) {
      return false;
    }
    if (readonlyDataSourceList.value.length > 0) {
      if (readonlyDataSourceList.value.some((ds) => !ds.host)) {
        return false;
      }
    }
    return !!basicInfo.value.title.trim();
  }
  return checkDataSource([
    adminDataSource.value,
    ...readonlyDataSourceList.value,
  ]);
});

// getOriginalEditState returns the origin instance data including
// basic information, admin data source and read-only data source.
const getOriginalEditState = () => {
  return {
    basicInfo: extractBasicInfo(instance.value),
    dataSources: extractDataSourceEditState(instance.value).dataSources,
  };
};

const confirmContinueWithConnectionFailure = (message: string) => {
  const d = defer<boolean>();
  $d.warning({
    title: t("common.warning"),
    content: t("instance.unable-to-connect", [message]),
    contentClass: "whitespace-pre-wrap",
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.continue-anyway"),
    onNegativeClick: () => {
      d.resolve(false);
    },
    onPositiveClick: () => {
      d.resolve(true);
    },
  });
  return d.promise;
};

const tryCreate = async () => {
  const editingDS = adminDataSource.value;
  const testResult = await testConnection(editingDS, /* silent */ true);
  if (testResult.success) {
    doCreate();
  } else {
    const confirmed = await confirmContinueWithConnectionFailure(
      testResult.message
    );
    if (confirmed) {
      doCreate();
    }
  }
};

const hasExternalSecretFeature = computed(() =>
  subscriptionStore.hasFeature(PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER)
);
const checkExternalSecretFeature = (dataSources: DataSource[]) => {
  if (hasExternalSecretFeature.value) {
    return true;
  }

  return dataSources.every((ds) => {
    return !ds.externalSecret && !/^{{.+}}$/.test(ds.password);
  });
};

const checkRODataSourceFeature = (instance: Instance) => {
  // This is to
  // - Disallow creating any new RO data sources
  // - Disallow updating any existed RO data sources
  // if feature is not available.

  // Early pass if the feature is available.
  if (hasReadonlyReplicaFeature.value) {
    return true;
  }

  if (readonlyDataSourceList.value.length === 0) {
    // Not creating or editing any RO data source
    return true;
  }

  const checkOne = (ds: EditDataSource) => {
    if (ds.pendingCreate) {
      // Disallowed to create any new RO data sources
      return false;
    } else {
      const editing = extractDataSourceFromEdit(instance.engine, ds);
      const original = instance.dataSources.find((d) => d.id === ds.id);
      if (original) {
        const updateMask = calcDataSourceUpdateMask(editing, original, ds);
        // Disallowed to update any existed RO data source
        if (updateMask.length > 0) {
          return false;
        }
      }
    }
    return true;
  };
  // Need to check all RO data sources
  return readonlyDataSourceList.value.every(checkOne);
};

// We will also create the database * denoting all databases
// and its RW data source. The username, password is actually
// stored in that data source object instead of in the instance self.
// Conceptually, data source is the proper place to store connection info (thinking of DSN)
const doCreate = async () => {
  if (!isCreating.value) {
    return;
  }

  if (!checkExternalSecretFeature(pendingCreateInstance.value.dataSources)) {
    missingFeature.value = PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER;
    return;
  }

  state.value.isRequesting = true;
  try {
    const createdInstance = await instanceV1Store.createInstance(
      pendingCreateInstance.value
    );
    if (props.onCreated) {
      props.onCreated(createdInstance);
    } else {
      router.push(`/${createdInstance.name}`);
      events.emit("dismiss");
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-created-instance-createdinstance-name", [
        createdInstance.title,
      ]),
    });
  } finally {
    state.value.isRequesting = false;
  }
};

const updateEditState = (instance: Instance) => {
  basicInfo.value = extractBasicInfo(instance);
  const updatedEditState = extractDataSourceEditState(instance);
  dataSourceEditState.value.dataSources = updatedEditState.dataSources;
  if (
    updatedEditState.dataSources.findIndex(
      (ds) => ds.id === dataSourceEditState.value.editingDataSourceId
    ) < 0
  ) {
    // The original selected data source id is no-longer in the latest data source list
    dataSourceEditState.value.editingDataSourceId =
      updatedEditState.editingDataSourceId;
  }
};

const doUpdate = async () => {
  const inst = instance.value;
  if (!inst) {
    return;
  }
  if (!checkRODataSourceFeature(inst)) {
    missingFeature.value = PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION;
    return;
  }

  if (!checkExternalSecretFeature([adminDataSource.value])) {
    missingFeature.value = PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER;
    return;
  }

  if (
    !checkExternalSecretFeature([
      adminDataSource.value,
      ...readonlyDataSourceList.value,
    ])
  ) {
    missingFeature.value = PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER;
    return;
  }

  // When clicking **Update** we may have more than one thing to do (if needed)
  // 1. Patch the instance itself.
  // 2. Update the admin datasource.
  // 3. Create OR update read-only data source(s).

  const pendingRequestRunners: (() => Promise<unknown>)[] = [];

  const maybeQueueUpdateInstanceBasicInfo = () => {
    const currentLabels = convertKVListToLabels(labelKVList.value, false);
    const instancePatch = create(InstanceSchema, {
      ...instance,
      ...basicInfo.value,
      labels: currentLabels,
    });
    const updateMask: string[] = [];
    if (instancePatch.title !== inst.title) {
      updateMask.push("title");
    }
    if (instancePatch.externalLink !== inst.externalLink) {
      updateMask.push("external_link");
    }
    if (instancePatch.activation !== inst.activation) {
      updateMask.push("activation");
    }
    if (instancePatch.environment !== inst.environment) {
      updateMask.push("environment");
    }
    if (
      Number(instancePatch.syncInterval?.seconds || 0n) !==
      Number(inst.syncInterval?.seconds || 0n)
    ) {
      updateMask.push("sync_interval");
    }
    if (!isEqual(instancePatch.syncDatabases, inst.syncDatabases)) {
      updateMask.push("sync_databases");
    }
    if (!isEqual(instancePatch.labels, inst.labels)) {
      updateMask.push("labels");
    }
    if (updateMask.length === 0) {
      return;
    }
    pendingRequestRunners.push(() =>
      instanceV1Store.updateInstance(instancePatch, updateMask).then(() => {
        if (updateMask.includes("sync_databases")) {
          return refreshInstanceDatabases(instancePatch.name);
        }
      })
    );
  };
  const refreshInstanceDatabases = async (instance: string) => {
    await instanceV1Store.syncInstance(instance, true);
    databaseStore.removeCacheByInstance(instance);
  };
  /**
   * @returns true if blocked by connection testing failure
   */
  const maybeQueueUpdateDataSource = async (
    editing: DataSource,
    original: DataSource | undefined,
    editState: EditDataSource
  ) => {
    if (!original) return;
    const updateMask = calcDataSourceUpdateMask(editing, original, editState);
    if (updateMask.length === 0) {
      return;
    }
    const testResult = await testConnection(editState, /* silent */ true);
    if (!testResult.success) {
      const continueAnyway = await confirmContinueWithConnectionFailure(
        testResult.message
      );
      if (!continueAnyway) {
        return true;
      }
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
    const editing = extractDataSourceFromEdit(
      inst.engine,
      adminDataSource.value
    );
    return await maybeQueueUpdateDataSource(
      editing,
      original,
      adminDataSource.value
    );
  };
  /**
   * @returns true if blocked by connection testing failure
   */
  const maybeQueueUpsertReadonlyDataSources = async () => {
    if (readonlyDataSourceList.value.length === 0) {
      // Nothing to do, don't block
      return false;
    }
    // Upsert readonly data sources one by one
    for (let i = 0; i < readonlyDataSourceList.value.length; i++) {
      const editing = readonlyDataSourceList.value[i];
      const patch = extractDataSourceFromEdit(inst.engine, editing);
      if (editing.pendingCreate) {
        const testResult = await testConnection(editing, /* silent */ true);
        if (!testResult.success) {
          const continueAnyway = await confirmContinueWithConnectionFailure(
            testResult.message
          );
          if (!continueAnyway) {
            return true;
          }
        }

        pendingRequestRunners.push(() =>
          instanceV1Store.createDataSource({
            instance: inst.name,
            dataSource: patch,
          })
        );
      } else {
        const original = inst.dataSources.find((ds) => ds.id === editing.id);
        const blocked = await maybeQueueUpdateDataSource(
          patch,
          original,
          editing
        );
        if (blocked) {
          return true;
        }
      }
    }
  };

  // prepare pending request runners
  await maybeQueueUpdateInstanceBasicInfo();
  if (await maybeQueueUpdateAdminDataSource()) {
    // blocked
    return;
  }
  if (await maybeQueueUpsertReadonlyDataSources()) {
    // blocked
    return;
  }

  if (pendingRequestRunners.length === 0) {
    return;
  }

  state.value.isRequesting = true;
  try {
    // Send requests one-by-one
    for (let i = 0; i < pendingRequestRunners.length; i++) {
      const runner = pendingRequestRunners[i];
      await runner();
    }

    const updatedInstance = instanceV1Store.getInstanceByName(inst.name);
    updateEditState(updatedInstance);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-updated-instance-instance-name", [
        updatedInstance.title,
      ]),
    });

    if (props.onUpdated) {
      props.onUpdated(updatedInstance);
    }
  } finally {
    state.value.isRequesting = false;
  }
};

const cancel = () => {
  events.emit("dismiss");
};
</script>
