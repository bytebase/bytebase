<template>
  <template v-if="context.status === 'EXECUTING'">
    <div
      class="w-full h-full flex flex-col justify-center items-center text-sm gap-y-1 bg-white/80 dark:bg-black/80"
    >
      <div class="flex items-center gap-x-1">
        <BBSpin :size="20" class="mr-1" />
        <span>{{ $t("sql-editor.executing-query") }}</span>
        <span>-</span>
        <!-- use mono font to prevent the UI jitters frequently -->
        <span class="font-mono">{{ queryElapsedTime }}</span>
      </div>
      <div>
        <NButton size="small" @click="cancelQuery">
          {{ $t("common.cancel") }}
        </NButton>
      </div>
    </div>
  </template>
  <template v-else-if="context.status === 'CANCELLED'">
    <div
      class="w-full h-full flex flex-col justify-center items-center text-sm gap-y-1 bg-white/80 dark:bg-black/80"
    >
      <NButton size="small" @click="() => execQuery(context.params)">
        {{ $t("sql-editor.execute-query") }}
      </NButton>
    </div>
  </template>
  <ResultViewV1
    v-else
    class="w-full h-auto grow"
    :execute-params="context.params"
    :database="database"
    :context-id="context.id"
    :result-set="context.resultSet"
    @execute="execQuery"
  />
</template>

<script lang="tsx" setup>
import { useIntervalFn } from "@vueuse/core";
import { NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { useSQLEditorTabStore } from "@/store";
import type {
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
} from "@/types";
import type { ComposedDatabase } from "@/types";
import { ResultViewV1 } from "../../EditorCommon/";

const props = defineProps<{
  database: ComposedDatabase;
  context: SQLEditorDatabaseQueryContext;
}>();

const executeSQL = useExecuteSQL();
const tabStore = useSQLEditorTabStore();

const loading = computed(() => props.context.status === "EXECUTING");

// Update every 1 second instead of every frame
const currentTimestampMS = ref(Date.now());
const { pause, resume } = useIntervalFn(() => {
  currentTimestampMS.value = Date.now();
}, 1000);

// Start/stop the timer based on loading state
watch(
  loading,
  (isLoading) => {
    if (isLoading) {
      currentTimestampMS.value = Date.now();
      resume();
    } else {
      pause();
    }
  },
  { immediate: true }
);

const queryElapsedTime = computed(() => {
  if (!loading.value) {
    return "";
  }
  const beginMS = props.context.beginTimestampMS;
  if (!beginMS) {
    return "";
  }
  const elapsedMS = currentTimestampMS.value - beginMS;
  return `${(elapsedMS / 1000).toFixed(1)}s`;
});

const cancelQuery = () => {
  props.context.abortController?.abort();
  tabStore.updateDatabaseQueryContext({
    database: props.database.name,
    contextId: props.context.id,
    context: {
      status: "CANCELLED",
    },
  });
};

watch(
  [() => props.database, () => props.context.status],
  async ([database, status]) => {
    if (status === "PENDING") {
      await executeSQL.runQuery(database, props.context);
    }
  },
  { immediate: true, deep: true }
);

const execQuery = async (params: SQLEditorQueryParams) => {
  const context = tabStore.updateDatabaseQueryContext({
    database: props.database.name,
    contextId: props.context.id,
    context: {
      params,
    },
  });
  if (!context) {
    return;
  }
  await executeSQL.runQuery(props.database, context);
};
</script>
