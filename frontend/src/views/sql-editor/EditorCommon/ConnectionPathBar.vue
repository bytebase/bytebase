<template>
  <div
    v-if="!tabStore.isDisconnected"
    class="w-full py-1 px-4 flex justify-between items-center border-b"
    :class="[isProtectedEnvironment && 'bg-yellow-50']"
  >
    <div :class="[isProtectedEnvironment ? 'text-yellow-700' : 'text-main']">
      <template v-if="isProtectedEnvironment">
        {{ $t("sql-editor.sql-execute-in-protected-environment") }}
      </template>
    </div>

    <div class="action-right flex justify-end items-center">
      <NPopover
        v-if="selectedInstance.id !== UNKNOWN_ID && !hasReadonlyDataSource"
        trigger="hover"
      >
        <template #trigger>
          <heroicons-outline:exclamation
            class="h-6 w-6 flex-shrink-0 mr-2"
            :class="[
              isProtectedEnvironment ? 'text-yellow-500' : 'text-yellow-500',
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

      <NPopover trigger="hover" placement="bottom" :show-arrow="false">
        <template #trigger>
          <label class="flex items-center text-sm space-x-1">
            <div
              v-if="selectedInstance.id !== UNKNOWN_ID"
              class="flex items-center"
            >
              <span class="">{{ selectedInstance.environment.name }}</span>
              <ProtectedEnvironmentIcon
                :environment="selectedInstance.environment"
                class="ml-1"
                :class="[isProtectedEnvironment && '~!text-yellow-700']"
              />
            </div>
            <div
              v-if="selectedInstance.id !== UNKNOWN_ID"
              class="flex items-center"
            >
              <span class="mx-2">/</span>
              <InstanceEngineIcon :instance="selectedInstance" show-status />
              <span class="ml-2">{{ selectedInstance.name }}</span>
            </div>
            <div
              v-if="selectedDatabase.id !== UNKNOWN_ID"
              class="flex items-center"
            >
              <span class="mx-2">/</span>
              <heroicons-outline:database />
              <span class="ml-2">{{ selectedDatabase.name }}</span>
            </div>
          </label>
        </template>
        <section>
          <div class="space-y-2">
            <div
              v-if="selectedInstance.id !== UNKNOWN_ID"
              class="flex flex-col"
            >
              <h1 class="text-gray-400">{{ $t("common.environment") }}</h1>
              <span class="flex items-center">
                {{ selectedInstance.environment.name }}
                <ProtectedEnvironmentIcon
                  :environment="selectedInstance.environment"
                  class="ml-1"
                />
              </span>
            </div>
            <div
              v-if="selectedInstance.id !== UNKNOWN_ID"
              class="flex flex-col"
            >
              <h1 class="text-gray-400">{{ $t("common.instance") }}</h1>
              <span>{{ selectedInstance.name }}</span>
            </div>
            <div
              v-if="selectedDatabase.id !== UNKNOWN_ID"
              class="flex flex-col"
            >
              <h1 class="text-gray-400">{{ $t("common.database") }}</h1>
              <span>{{ selectedDatabase.name }}</span>
            </div>
          </div>
        </section>
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NPopover } from "naive-ui";
import { useRouter } from "vue-router";

import { useTabStore, useInstanceById, useDatabaseById } from "@/store";
import { UNKNOWN_ID } from "@/types";
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
const isProtectedEnvironment = computed(() => {
  const instance = selectedInstance.value;
  return instance.environment.tier === "PROTECTED";
});

const hasReadonlyDataSource = computed(() => {
  for (const ds of selectedInstance.value.dataSourceList) {
    if (ds.type === "RO") {
      return true;
    }
  }
  return false;
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
