<template>
  <div class="w-full flex justify-between items-center p-4 border-b">
    <div class="action-left space-x-2 flex items-center">
      <NButton
        type="primary"
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleRunQuery"
      >
        <mdi:play class="h-5 w-5" /> {{ $t("common.run") }} (⌘+⏎)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleExplainQuery"
      >
        <mdi:play class="h-5 w-5" /> Explain (⌘+E)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleFormatSQL"
      >
        {{ $t("sql-editor.format") }} (⇧+⌥+F)
      </NButton>
    </div>
    <div class="action-right space-x-2 flex justify-end items-center">
      <NPopover
        v-if="selectedInstance.id !== UNKNOWN_ID && !hasReadonlyDataSource"
        trigger="hover"
      >
        <template #trigger>
          <heroicons-outline:exclamation
            class="h-6 w-6 text-yellow-400 flex-shrink-0 mr-2"
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

      <NPopover trigger="hover" placement="bottom-center" :show-arrow="false">
        <template #trigger>
          <label class="flex items-center text-sm space-x-1">
            <div
              v-if="selectedInstance.id !== UNKNOWN_ID"
              class="flex items-center"
            >
              <span class="ml-2">{{ selectedInstance.environment.name }}</span>
              <ProtectedEnvironmentIcon
                :environment="selectedInstance.environment"
                class="ml-1"
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
      <NButton
        secondary
        strong
        type="primary"
        :disabled="!allowSave"
        @click="() => emit('save-sheet')"
      >
        <carbon:save class="h-5 w-5" /> &nbsp; {{ $t("common.save") }} (⌘+S)
      </NButton>
      <NPopover trigger="click" placement="bottom-end" :show-arrow="false">
        <template #trigger>
          <NButton
            :disabled="
              isEmptyStatement ||
              tabStore.isDisconnected ||
              !tabStore.currentTab.isSaved
            "
          >
            <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }}
          </NButton>
        </template>
        <template #default>
          <SharePopover />
        </template>
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, defineEmits } from "vue";
import { useRouter } from "vue-router";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import {
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
  useInstanceById,
  useDatabaseById,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { instanceSlug } from "@/utils/slug";
import SharePopover from "./SharePopover.vue";
import ProtectedEnvironmentIcon from "@/components/Environment/ProtectedEnvironmentIcon.vue";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
}>();

const router = useRouter();
const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const connection = computed(() => tabStore.currentTab.connection);

const isEmptyStatement = computed(
  () => !tabStore.currentTab || tabStore.currentTab.statement === ""
);
const isExecutingSQL = computed(() => tabStore.currentTab.isExecutingSQL);
const selectedInstance = useInstanceById(
  computed(() => connection.value.instanceId)
);
const selectedInstanceEngine = computed(() => {
  return instanceStore.formatEngine(selectedInstance.value);
});
const selectedDatabase = useDatabaseById(
  computed(() => connection.value.databaseId)
);

const hasReadonlyDataSource = computed(() => {
  for (const ds of selectedInstance.value.dataSourceList) {
    if (ds.type === "RO") {
      return true;
    }
  }
  return false;
});

const allowSave = computed(() => {
  if (isEmptyStatement.value) {
    return false;
  }
  if (tabStore.currentTab.isSaved) {
    return false;
  }
  // Temporarily disable saving and sharing if we are connected to an instance
  // but not a database.
  if (tabStore.currentTab.connection.databaseId === UNKNOWN_ID) {
    return false;
  }
  return true;
});

const { execute } = useExecuteSQL();

const handleRunQuery = async () => {
  const currentTab = tabStore.currentTab;
  const statement = currentTab.statement;
  const selectedStatement = currentTab.selectedStatement;
  const query = selectedStatement || statement;
  await execute(query, { databaseType: selectedInstanceEngine.value });
};

const handleExplainQuery = () => {
  const currentTab = tabStore.currentTab;
  const statement = currentTab.statement;
  const selectedStatement = currentTab.selectedStatement;
  const query = selectedStatement || statement;
  execute(
    query,
    { databaseType: selectedInstanceEngine.value },
    { explain: true }
  );
};

const gotoInstanceDetailPage = () => {
  const route = router.resolve({
    name: "workspace.instance.detail",
    params: {
      instanceSlug: instanceSlug(selectedInstance.value),
    },
  });
  window.open(route.href);
};

const handleFormatSQL = () => {
  sqlEditorStore.setShouldFormatContent(true);
};
</script>
