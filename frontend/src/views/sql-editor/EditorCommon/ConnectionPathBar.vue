<template>
  <div
    class="w-full flex flex-col sm:flex-row sm:flex-wrap lg:flex-nowrap lg:justify-between items-start bg-white overflow-hidden"
  >
    <div
      v-if="currentTab && !isDisconnected"
      class="flex justify-start items-center h-8 px-4 whitespace-nowrap shrink-0 gap-x-4"
    >
      <NPopover
        v-if="instance.uid !== String(UNKNOWN_ID)"
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
            {{ environment.title }}
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
          v-if="instance.uid !== String(UNKNOWN_ID)"
          class="flex items-center"
        >
          <InstanceV1EngineIcon :instance="instance" show-status />
          <span class="ml-2">{{ instance.title }}</span>
        </div>
        <div
          v-if="database.uid !== String(UNKNOWN_ID)"
          class="flex items-center"
        >
          <span class="mx-2">
            <heroicons-solid:chevron-right
              class="flex-shrink-0 h-4 w-4 text-control-light"
            />
          </span>
          <heroicons-outline:database />
          <span class="ml-2">{{ database.databaseName }}</span>
        </div>

        <div
          v-if="showBatchQuerySelector"
          class="relative ml-2 flex items-center"
        >
          <BatchQueryDatabasesSelector />
        </div>
      </label>

      <ReadonlyDatasourceHint :instance="instance" />
    </div>

    <div
      v-else
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
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import BatchQueryDatabasesSelector from "./BatchQueryDatabasesSelector.vue";
import ReadonlyDatasourceHint from "./ReadonlyDatasourceHint.vue";

const tabStore = useSQLEditorTabStore();
const { currentTab, isDisconnected } = storeToRefs(tabStore);

const { instance, database, environment } =
  useConnectionOfCurrentSQLEditorTab();

const isProductionEnvironment = computed(() => {
  if (!currentTab.value) {
    return false;
  }
  if (isDisconnected.value) {
    return false;
  }

  return environment.value.tier === EnvironmentTier.PROTECTED;
});

const showBatchQuerySelector = computed(() => {
  const tab = currentTab.value;
  return (
    tab &&
    // Only show entry when user selected a database.
    database.value.uid !== String(UNKNOWN_ID) &&
    // TODO(steven): implement batch query in admin mode.
    tab.mode !== "ADMIN"
  );
});
</script>
