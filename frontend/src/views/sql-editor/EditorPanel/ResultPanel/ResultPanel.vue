<template>
  <div
    class="relative w-full h-full flex flex-col justify-start items-start z-10 overflow-x-hidden"
  >
    <BatchQuerySelect v-model:selected-database="selectedDatabase" />
    <NTabs
      v-if="selectedDatabase && queryContexts"
      type="card"
      size="small"
      class="flex-1 flex flex-col overflow-hidden px-2"
      :class="isBatchQuery ? 'pt-0' : 'pt-2'"
      style="--n-tab-padding: 4px 12px"
      v-model:value="selectedTab"
    >
      <NTabPane
        v-for="(context, i) in queryContexts"
        :key="i"
        :name="context.id"
        class="flex-1 flex flex-col overflow-hidden"
      >
        <template #tab>
          <NTooltip>
            <template #trigger>
              <div class="flex items-center space-x-2">
                <span>{{ tabName(context) }}</span>
                <Info
                  v-if="context.resultSet?.error"
                  class="text-yellow-600 w-4 h-auto"
                />
                <BBSpin v-if="context.status === 'EXECUTING'" :size="10" />
              </div>
            </template>
            {{ context.params.statement }}
          </NTooltip>
        </template>
        <DatabaseQueryContext
          class="w-full h-auto grow"
          :database="selectedDatabase"
          :context="context"
        />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head } from "lodash-es";
import { Info } from "lucide-vue-next";
import { NTabs, NTabPane, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { useSQLEditorTabStore } from "@/store";
import type { ComposedDatabase, SQLEditorDatabaseQueryContext } from "@/types";
import BatchQuerySelect from "./BatchQuerySelect.vue";
import DatabaseQueryContext from "./DatabaseQueryContext.vue";

const selectedDatabase = ref<ComposedDatabase>();
const tabStore = useSQLEditorTabStore();
const selectedTab = ref<string>();
const { t } = useI18n();

const isBatchQuery = computed(
  () =>
    Array.from(tabStore.currentTab?.databaseQueryContexts?.keys() || [])
      .length > 1
);

const queryContexts = computed(() => {
  const contexts = tabStore.currentTab?.databaseQueryContexts?.get(
    selectedDatabase.value?.name ?? ""
  );
  return contexts;
});

const tabName = (context: SQLEditorDatabaseQueryContext) => {
  switch (context.status) {
    case "PENDING":
      return t("sql-editor.pending-query");
    case "EXECUTING":
      return t("sql-editor.executing-query");
    default:
      return dayjs(context.beginTimestampMS).format("YYYY-MM-DD HH:mm:ss");
  }
};

watch(
  () => head(queryContexts.value)?.id,
  (id) => {
    selectedTab.value = id;
  },
  { immediate: true }
);
</script>
