<template>
  <div class="w-full block lg:flex justify-between items-start bg-white">
    <div
      v-if="!tabStore.isDisconnected"
      class="flex justify-start items-center h-8 px-4 whitespace-nowrap overflow-x-auto"
    >
      <NPopover v-if="showReadonlyDatasourceHint" trigger="hover">
        <template #trigger>
          <heroicons-outline:information-circle
            class="h-5 w-5 flex-shrink-0 mr-2 text-info"
          />
        </template>
        <p class="py-1">
          {{ $t("instance.no-read-only-data-source-warn") }}
          <span
            class="underline text-accent cursor-pointer hover:opacity-80"
            @click="gotoInstanceDetailPage"
          >
            {{ $t("sql-editor.create-read-only-data-source") }}
          </span>
        </p>
      </NPopover>

      <label class="flex items-center text-sm space-x-1">
        <div
          v-if="selectedInstance.uid !== String(UNKNOWN_ID)"
          class="flex items-center"
        >
          <ProductionEnvironmentV1Icon
            :environment="selectedEnvironment"
            :class="[isProductionEnvironment && '~!text-yellow-700']"
          />
          <span class="ml-1">{{ selectedEnvironment.title }}</span>
        </div>
        <div
          v-if="selectedInstance.uid !== String(UNKNOWN_ID)"
          class="flex items-center"
        >
          <span class="mx-2">
            <heroicons-solid:chevron-right
              class="flex-shrink-0 h-4 w-4 text-control-light"
            />
          </span>
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
      </label>
    </div>

    <div
      v-if="isProductionEnvironment"
      class="flex justify-start items-center py-1 sm:py-0 sm:h-6 px-4 sm:rounded-bl text-white text-sm bg-error"
    >
      {{ $t("sql-editor.sql-execute-in-production-environment") }}
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
import {
  InstanceV1EngineIcon,
  ProductionEnvironmentV1Icon,
} from "@/components/v2";
import { useTabStore, useDatabaseV1ByUID, useInstanceV1ByUID } from "@/store";
import { TabMode, UNKNOWN_ID } from "@/types";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import { DataSourceType } from "@/types/proto/v1/instance_service";
import { instanceV1Slug } from "@/utils";

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

const gotoInstanceDetailPage = () => {
  const route = router.resolve({
    name: "workspace.instance.detail",
    params: {
      instanceSlug: instanceV1Slug(selectedInstance.value),
    },
  });
  window.open(route.href);
};
</script>
