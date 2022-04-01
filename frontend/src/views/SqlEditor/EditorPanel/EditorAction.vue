<template>
  <div class="editor-action">
    <div class="action-left space-x-2 flex items-center">
      <NButton
        type="primary"
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleRunQuery"
      >
        <mdi:play class="h-5 w-5" /> {{ $t("common.run") }} (⌘+⏎)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleExplainQuery"
      >
        <mdi:play class="h-5 w-5" /> Explain (⌘+E)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleFormatSQL"
      >
        {{ $t("sql-editor.format") }} (⇧+⌥+F)
      </NButton>
    </div>
    <div class="action-right space-x-2 flex justify-end items-center">
      <NPopover
        v-if="
          connectionContext.instanceId !== UNKNOWN_ID && !hasReadonlyDataSource
        "
        trigger="hover"
      >
        <template #trigger>
          <heroicons-outline:exclamation
            class="h-6 w-6 text-yellow-400 flex-shrink-0 mr-2"
          />
        </template>
        <p class="py-1">
          {{ $t("instance.no-read-only-data-source-warn") }}
          <NButton
            class="text-base underline text-accent"
            text
            @click="gotoInstanceDetailPage"
          >
            {{ $t("sql-editor.create-read-only-data-source") }}
          </NButton>
        </p>
      </NPopover>
      <NPopover trigger="hover" placement="bottom-center" :show-arrow="false">
        <template #trigger>
          <label class="flex items-center text-sm space-x-1">
            <div class="flex items-center">
              <InstanceEngineIcon
                v-if="connectionContext.instanceId !== UNKNOWN_ID"
                :instance="selectedInstance"
                show-status
              />
              <span class="ml-2">{{ connectionContext.instanceName }}</span>
            </div>
            <div
              v-if="connectionContext.databaseName"
              class="flex items-center"
            >
              &nbsp; / &nbsp;
              <heroicons-outline:database />
              <span class="ml-2">{{ connectionContext.databaseName }}</span>
            </div>
            <div v-if="connectionContext.tableName" class="flex items-center">
              &nbsp; / &nbsp;
              <heroicons-outline:table />
              <span class="ml-2">{{ connectionContext.tableName }}</span>
            </div>
          </label>
        </template>
        <section>
          <div class="space-y-2">
            <div v-if="connectionContext.instanceName" class="flex flex-col">
              <h1 class="text-gray-400">{{ $t("common.instance") }}:</h1>
              <span>{{ connectionContext.instanceName }}</span>
            </div>
            <div v-if="connectionContext.databaseName" class="flex flex-col">
              <h1 class="text-gray-400">{{ $t("common.database") }}:</h1>
              <span>{{ connectionContext.databaseName }}</span>
            </div>
            <div v-if="connectionContext.tableName" class="flex flex-col">
              <h1 class="text-gray-400">{{ $t("common.table") }}:</h1>
              <span>{{ connectionContext.tableName }}</span>
            </div>
          </div>
        </section>
      </NPopover>
      <NButton
        secondary
        strong
        type="primary"
        :disabled="isEmptyStatement || tabStore.currentTab.isSaved"
        @click="() => emit('save-sheet')"
      >
        <carbon:save class="h-5 w-5" /> &nbsp; {{ $t("common.save") }} (⌘+S)
      </NButton>
      <NPopover
        trigger="click"
        placement="bottom-end"
        :show-arrow="false"
        :show="isShowSharePopover"
        :on-clickoutside="handleClickoutside"
      >
        <template #trigger>
          <NButton
            :disabled="
              isEmptyStatement || isDisconnected || !tabStore.currentTab.isSaved
            "
            @click.stop.prevent="toggleSharePopover"
          >
            <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }}
          </NButton>
        </template>
        <SharePopover @close="isShowSharePopover = false" />
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, defineEmits } from "vue";
import {
  useNamespacedState,
  useNamespacedGetters,
  useNamespacedActions,
} from "vuex-composition-helpers";
import { useStore } from "vuex";

import { useTabStore } from "@/store/pinia-modules/tab";
import {
  SqlEditorState,
  SqlEditorGetters,
  SqlEditorActions,
  UNKNOWN_ID,
  Instance,
} from "@/types";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import SharePopover from "./SharePopover.vue";
import { useRouter } from "vue-router";
import { instanceSlug } from "../../../utils/slug";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
}>();

const store = useStore();
const router = useRouter();
const tabStore = useTabStore();

const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);
const { isDisconnected } = useNamespacedGetters<SqlEditorGetters>("sqlEditor", [
  "isDisconnected",
]);

const { setShouldFormatContent } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setShouldFormatContent"]
);

const isShowSharePopover = ref(false);

const isEmptyStatement = computed(
  () => !tabStore.currentTab || tabStore.currentTab.statement === ""
);
const selectedInstance = computed<Instance>(() => {
  const ctx = connectionContext.value;
  return store.getters["instance/instanceById"](ctx.instanceId);
});
const selectedInstanceEngine = computed(() => {
  return store.getters["instance/instanceFormatedEngine"](
    selectedInstance.value
  ) as string;
});

const hasReadonlyDataSource = computed(() => {
  for (const ds of selectedInstance.value.dataSourceList) {
    if (ds.type === "RO") {
      return true;
    }
  }
  return false;
});

const { execute, state: executeState } = useExecuteSQL();

const handleRunQuery = () => {
  execute({ databaseType: selectedInstanceEngine.value });
};

const handleExplainQuery = () => {
  execute({ databaseType: selectedInstanceEngine.value }, { explain: true });
};

const gotoInstanceDetailPage = () => {
  router.push({
    name: "workspace.instance.detail",
    params: {
      instanceSlug: instanceSlug(selectedInstance.value),
    },
  });
};

const handleClickoutside = (e: any) => {
  isShowSharePopover.value = false;
};

const toggleSharePopover = () => {
  isShowSharePopover.value = !isShowSharePopover.value;
};

const handleFormatSQL = () => {
  setShouldFormatContent(true);
};
</script>

<style scoped>
.editor-action {
  @apply w-full flex justify-between items-center p-2;
}
</style>
