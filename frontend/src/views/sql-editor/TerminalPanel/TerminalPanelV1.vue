<template>
  <div
    class="flex h-full w-full flex-col justify-start items-start overflow-hidden bg-dark-bg"
  >
    <EditorAction @execute="handleExecute" @clear-screen="handleClearScreen" />

    <ConnectionPathBar />

    <div
      v-if="!tabStore.isDisconnected"
      ref="queryListContainerRef"
      class="w-full flex-1 overflow-y-auto bg-dark-bg"
    >
      <div
        ref="queryListRef"
        class="w-full flex flex-col"
        :data-height="queryListHeight"
      >
        <div v-for="query in queryList" :key="query.id" class="relative">
          <CompactSQLEditor
            v-model:sql="query.sql"
            class="min-h-[2rem]"
            :class="[
              isEditableQueryItem(query) ? 'active-editor' : 'read-only-editor',
            ]"
            :readonly="!isEditableQueryItem(query)"
            @execute="handleExecute"
            @history="handleHistory"
            @clear-screen="handleClearScreen"
          />
          <ResultViewV1
            v-if="query.params && query.resultSet"
            class="max-h-[20rem] flex-1 flex flex-col overflow-hidden"
            :execute-params="query.params"
            :result-set="query.resultSet"
            :loading="query.status === 'RUNNING'"
            :dark="true"
          />

          <div
            v-if="query.resultSet?.error"
            class="p-2 pb-1 text-md font-normal text-[var(--color-matrix-green-hover)]"
          >
            {{ $t("sql-editor.connection-lost") }}
          </div>

          <div
            v-if="query.status === 'RUNNING'"
            class="absolute inset-0 bg-black/20 flex justify-center items-center gap-2"
          >
            <BBSpin />
            <div
              v-if="query === currentQuery && expired"
              class="text-gray-400 cursor-pointer hover:underline select-none"
              @click="handleCancelQuery"
            >
              {{ $t("common.cancel") }}
            </div>
          </div>
        </div>
      </div>
    </div>
    <ConnectionHolder v-else />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { computed, ref, unref, watch } from "vue";
import { useTabStore, useWebTerminalV1Store } from "@/store";
import { ExecuteConfig, ExecuteOption, WebTerminalQueryItemV1 } from "@/types";
import {
  EditorAction,
  ConnectionPathBar,
  ConnectionHolder,
  ResultViewV1,
} from "../EditorCommon";
import CompactSQLEditor from "./CompactSQLEditor.vue";
import { useAttractFocus } from "./useAttractFocus";
import { useHistory } from "./useHistory";

const tabStore = useTabStore();
const webTerminalStore = useWebTerminalV1Store();

const queryState = computed(() => {
  return webTerminalStore.getQueryStateByTab(tabStore.currentTab);
});

const queryList = computed(() => {
  return unref(queryState.value.queryItemList);
});

const queryListContainerRef = ref<HTMLDivElement>();
const queryListRef = ref<HTMLDivElement>();

const currentQuery = computed(
  () => queryList.value[queryList.value.length - 1]
);

const { move: moveHistory } = useHistory();

const timer = computed(() => {
  return unref(queryState.value.timer);
});
const expired = computed(() => {
  return unref(timer.value.expired);
});

const isEditableQueryItem = (item: WebTerminalQueryItemV1): boolean => {
  return item === currentQuery.value && item.status === "IDLE";
};

const handleExecute = async (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  if (currentQuery.value.status !== "IDLE") {
    return;
  }

  // Prevent executing empty query;
  if (!query) {
    return;
  }

  queryState.value.controller.events.emit("query", { query, config, option });
};

const handleClearScreen = () => {
  const list = queryList.value;
  while (list.length > 1) {
    list.shift();
  }
};

const handleHistory = (direction: "up" | "down") => {
  if (currentQuery.value.status !== "IDLE") {
    return;
  }
  moveHistory(direction);
};

const handleCancelQuery = async () => {
  queryState.value.controller.abort();
};

const { height: queryListHeight } = useElementSize(queryListRef);

watch(queryListHeight, () => {
  // Always scroll to the bottom as if we are in a real terminal.
  requestAnimationFrame(() => {
    const container = queryListContainerRef.value;
    if (container) {
      container.scrollTo(0, container.scrollHeight);
    }
  });
});

useAttractFocus({
  excluded: [{ tag: "textarea", selector: ".active-editor" }],
  targetSelector: ".active-editor textarea",
});
</script>
