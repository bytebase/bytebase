<template>
  <div
    v-if="!tabStore.isDisconnected"
    class="w-full flex justify-between items-start"
  >
    <div class="flex justify-start items-center h-8 px-4 rounded-br bg-white">
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
          v-if="selectedInstance.id !== UNKNOWN_ID"
          class="flex items-center"
        >
          <span class="">{{ selectedInstance.environment.name }}</span>
          <ProductionEnvironmentIcon
            :environment="selectedInstance.environment"
            class="ml-1"
            :class="[isProductionEnvironment && '~!text-yellow-700']"
          />
        </div>
        <div
          v-if="selectedInstance.id !== UNKNOWN_ID"
          class="flex items-center"
        >
          <span class="mx-2">
            <heroicons-solid:chevron-right
              class="flex-shrink-0 h-4 w-4 text-control-light"
            />
          </span>
          <InstanceEngineIcon :instance="selectedInstance" show-status />
          <span class="ml-2">{{ selectedInstance.name }}</span>
        </div>
        <div
          v-if="selectedDatabase.id !== UNKNOWN_ID"
          class="flex items-center"
        >
          <span class="mx-2">
            <heroicons-solid:chevron-right
              class="flex-shrink-0 h-4 w-4 text-control-light"
            />
          </span>
          <heroicons-outline:database />
          <span class="ml-2">{{ selectedDatabase.name }}</span>
        </div>
      </label>
    </div>

    <div
      v-if="isProductionEnvironment"
      class="flex justify-start items-center h-8 px-4 rounded-bl text-white bg-error"
    >
      {{ $t("sql-editor.sql-execute-in-production-environment") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NPopover } from "naive-ui";
import { useRouter } from "vue-router";

import { useTabStore, useInstanceById, useDatabaseById } from "@/store";
import { TabMode, UNKNOWN_ID } from "@/types";
import { instanceSlug } from "@/utils";

const router = useRouter();
const tabStore = useTabStore();

const connection = computed(() => tabStore.currentTab.connection);

const selectedInstance = useInstanceById(
  computed(() => connection.value.instanceId)
);

const selectedDatabase = useDatabaseById(
  computed(() => connection.value.databaseId)
);

const isProductionEnvironment = computed(() => {
  const instance = selectedInstance.value;
  return instance.environment.tier === "PROTECTED";
});

const isAdminMode = computed(() => {
  return tabStore.currentTab.mode === TabMode.Admin;
});

const hasReadonlyDataSource = computed(() => {
  for (const ds of selectedInstance.value.dataSourceList) {
    if (ds.type === "RO") {
      return true;
    }
  }
  return false;
});

const showReadonlyDatasourceWarning = computed(() => {
  return (
    !isAdminMode.value &&
    selectedInstance.value.id !== UNKNOWN_ID &&
    !hasReadonlyDataSource.value
  );
});

const gotoInstanceDetailPage = () => {
  const route = router.resolve({
    name: "workspace.instance.detail",
    params: {
      instanceSlug: instanceSlug(selectedInstance.value),
    },
  });
  window.open(route.href);
};
</script>
