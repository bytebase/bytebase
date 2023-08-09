<template>
  <div
    v-if="!tabStore.isDisconnected"
    class="w-full block lg:flex justify-between items-start bg-white"
  >
    <div
      class="flex justify-start items-center h-8 px-4 whitespace-nowrap overflow-x-auto"
    >
      <NPopover v-if="showReadonlyDatasourceWarning" trigger="hover">
        <template #trigger>
          <heroicons-outline:exclamation
            class="h-6 w-6 flex-shrink-0 mr-2"
            :class="[
              isProductionEnvironment ? 'text-yellow-500' : 'text-yellow-500',
            ]"
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
          <span class="">{{ selectedInstance.environmentEntity.title }}</span>
          <ProductionEnvironmentV1Icon
            :environment="selectedInstance.environmentEntity"
            class="ml-1"
            :class="[isProductionEnvironment && '~!text-yellow-700']"
          />
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
      class="flex justify-start items-center py-1 sm:py-0 sm:h-8 px-4 sm:rounded-bl text-white bg-error"
    >
      {{ $t("sql-editor.sql-execute-in-production-environment") }}
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

const connection = computed(() => tabStore.currentTab.connection);

const { instance: selectedInstance } = useInstanceV1ByUID(
  computed(() => connection.value.instanceId)
);

const { database: selectedDatabaseV1 } = useDatabaseV1ByUID(
  computed(() => String(connection.value.databaseId))
);

const isProductionEnvironment = computed(() => {
  const instance = selectedInstance.value;
  return instance.environmentEntity.tier === EnvironmentTier.PROTECTED;
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

const showReadonlyDatasourceWarning = computed(() => {
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
