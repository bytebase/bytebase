<template>
  <div
    class="w-full flex flex-col sm:flex-row sm:flex-wrap lg:flex-nowrap lg:justify-between items-start bg-white overflow-hidden"
  >
    <div
      v-if="!tabStore.isDisconnected"
      class="flex justify-start items-center h-8 px-4 whitespace-nowrap shrink-0 gap-x-4"
    >
      <NPopover
        v-if="selectedInstance.uid !== String(UNKNOWN_ID)"
        :disabled="!isProductionEnvironment"
      >
        <template #trigger>
          <div
            class="inline-flex items-center px-2 border text-sm rounded-sm"
            :class="[
              isProductionEnvironment
                ? 'border-error text-error'
                : 'border-main text-main',
            ]"
          >
            {{ selectedEnvironment.title }}
          </div>
        </template>
        <template #default>
          <div class="max-w-[20rem]">
            {{ $t("sql-editor.sql-execute-in-production-environment") }}
          </div>
        </template>
      </NPopover>

      <label class="flex items-center text-sm space-x-1">
        <div
          v-if="selectedInstance.uid !== String(UNKNOWN_ID)"
          class="flex items-center"
        >
          <InstanceV1EngineIcon :instance="selectedInstance" show-status />
          <span class="ml-2">{{ selectedInstance.title }}</span>
        </div>
        <div
          v-if="selectedDatabaseV1.uid !== String(UNKNOWN_ID)"
          class="flex items-center"
        >
          <span class="mx-2">
            <heroicons-solid:chevron-right
              class="flex-shrink-0 h-4 w-4 text-control-light"
            />
          </span>
          <heroicons-outline:database />
          <span class="ml-2">{{ selectedDatabaseV1.databaseName }}</span>
        </div>

        <div
          v-if="showBatchQuerySelector"
          class="relative ml-2 flex items-center"
        >
          <BatchQueryDatabasesSelector />
        </div>
      </label>

      <ReadonlyDatasourceHint :instance="selectedInstance" />
    </div>

    <div
      v-if="tabStore.isDisconnected && !currentTab.sheetName"
      class="flex justify-start items-center h-8 px-4 whitespace-nowrap overflow-x-auto"
    >
      <div class="text-sm text-control">
        {{ $t("sql-editor.connection-holder") }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NPopover } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import EllipsisText from "@/components/EllipsisText.vue";
import {
  InstanceV1EngineIcon,
  ProductionEnvironmentV1Icon,
} from "@/components/v2";
import {
  useTabStore,
  useDatabaseV1ByUID,
  useInstanceV1ByUID,
  useCurrentUserV1,
} from "@/store";
import { TabMode, UNKNOWN_ID } from "@/types";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import { DataSourceType } from "@/types/proto/v1/instance_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV1, instanceV1Slug } from "@/utils";
import BatchQueryDatabasesSelector from "./BatchQueryDatabasesSelector.vue";
import ReadonlyDatasourceHint from "./ReadonlyDatasourceHint.vue";

const router = useRouter();
const tabStore = useTabStore();
const currentTab = computed(() => tabStore.currentTab);
const connection = computed(() => currentTab.value.connection);

const { instance: selectedInstance } = useInstanceV1ByUID(
  computed(() => connection.value.instanceId)
);

const { database: selectedDatabaseV1 } = useDatabaseV1ByUID(
  computed(() => String(connection.value.databaseId))
);

const selectedProject = computed(() => selectedDatabaseV1.value.projectEntity);

const selectedEnvironment = computed(() => {
  return connection.value.databaseId === `${UNKNOWN_ID}`
    ? selectedInstance.value.environmentEntity
    : selectedDatabaseV1.value.effectiveEnvironmentEntity;
});

const isProductionEnvironment = computed(() => {
  if (tabStore.isDisconnected) {
    return false;
  }

  return selectedEnvironment.value.tier === EnvironmentTier.PROTECTED;
});

const isAdminMode = computed(() => {
  return tabStore.currentTab.mode === TabMode.Admin;
});

const hasReadonlyDataSource = computed(() => {
  return (
    selectedInstance.value.dataSources.findIndex(
      (ds) => ds.type === DataSourceType.READ_ONLY
    ) !== -1
  );
});

const showReadonlyDatasourceHint = computed(() => {
  return (
    !isAdminMode.value &&
    selectedInstance.value.uid !== String(UNKNOWN_ID) &&
    !hasReadonlyDataSource.value
  );
});

const showBatchQuerySelector = computed(() => {
  return (
    selectedProject.value.tenantMode === TenantMode.TENANT_MODE_ENABLED &&
    // TODO(steven): implement batch query in admin mode.
    currentTab.value.mode !== TabMode.Admin
  );
});
</script>
