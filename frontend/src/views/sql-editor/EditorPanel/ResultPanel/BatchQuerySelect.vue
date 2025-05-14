<template>
  <div
    v-if="queriedDatabaseNames.length > 1"
    class="w-full flex flex-row justify-start items-center p-2 pb-0 gap-2 shrink-0"
  >
    <NTooltip v-if="showEmptySwitch">
      <template #trigger>
        <NButton
          tertiary
          size="small"
          :type="showEmpty ? 'primary' : 'default'"
          style="--n-padding: 6px"
          @click="showEmpty = !showEmpty"
        >
          <EyeIcon v-if="showEmpty" class="w-4 h-4" />
          <EyeOffIcon v-else class="w-4 h-4" />
        </NButton>
      </template>
      <template #default>
        {{ $t("sql-editor.batch-query.show-or-hide-empty-query-results") }}
      </template>
    </NTooltip>

    <NScrollbar x-scrollable content-class="pb-2">
      <div class="flex flex-row justify-start items-center gap-2">
        <NButton
          v-for="item in filteredItems"
          :key="item.database.name"
          secondary
          strong
          size="small"
          :type="'default'"
          :style="{
            ...getBackgroundColorRgb(item.database),
            borderTop: selectedDatabase === item.database ? '3px solid' : '',
          }"
          @click="$emit('update:selected-database', item.database)"
        >
          <RichDatabaseName :database="item.database" />
          <InfoIcon
            v-if="isDatabaseQueryFailed(item)"
            class="ml-1 text-yellow-600 w-4 h-auto"
          />
          <span
            v-if="isEmptyQueryItem(item)"
            class="text-control-placeholder italic ml-1"
          >
            ({{ $t("common.empty") }})
          </span>
          <XIcon
            class="ml-1 text-gray-400 w-4 h-auto hover:text-gray-600"
            @click.stop="handleCloseSingleResultView(item.database)"
          />
        </NButton>
      </div>
    </NScrollbar>
  </div>
</template>

<script setup lang="ts">
import { useLocalStorage } from "@vueuse/core";
import { head, last } from "lodash-es";
import { EyeIcon, EyeOffIcon, InfoIcon, XIcon } from "lucide-vue-next";
import { NButton, NTooltip, NScrollbar } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, watch } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useDatabaseV1Store, useSQLEditorTabStore } from "@/store";
import type { ComposedDatabase, SQLResultSetV1 } from "@/types";
import { hexToRgb } from "@/utils";

type BatchQueryItem = {
  database: ComposedDatabase;
  resultSet: SQLResultSetV1 | undefined;
};

const props = defineProps<{
  selectedDatabase: ComposedDatabase | undefined;
}>();

const emit = defineEmits<{
  (event: "update:selected-database", db: ComposedDatabase | undefined): void;
}>();

const tabStore = useSQLEditorTabStore();
const { currentTab: tab } = storeToRefs(tabStore);
const databaseStore = useDatabaseV1Store();
const showEmpty = useLocalStorage(
  "bb.sql-editor.batch-query.show-empty-result-sets",
  true
);

const queriedDatabaseNames = computed(() =>
  Array.from(tab.value?.queryContext?.results.keys() || [])
);

const items = computed(() => {
  return queriedDatabaseNames.value.map<BatchQueryItem>((name) => {
    const database = databaseStore.getDatabaseByName(name);
    // TODO(ed):
    const result = last(tab.value?.queryContext?.results.get(name));
    return { database, resultSet: result?.resultSet };
  });
});

const isEmptyQueryItem = (item: BatchQueryItem) => {
  if (!item.resultSet) {
    return true;
  }
  if (item.resultSet.error) {
    // Failed queries have empty result sets, but should not be recognized
    // as empty result sets.
    return false;
  }
  return item.resultSet.results.every((result) => result.rows.length === 0);
};

const filteredItems = computed(() => {
  if (showEmpty.value) {
    return items.value;
  }

  return items.value.filter((item) => !isEmptyQueryItem(item));
});

const showEmptySwitch = computed(() => {
  if (items.value.length <= 1) {
    return false;
  }
  return items.value.some((item) => isEmptyQueryItem(item));
});

const isDatabaseQueryFailed = (item: BatchQueryItem) => {
  // If there is any error in the result set, we consider the query failed.
  return (
    item.resultSet?.error ||
    item.resultSet?.results.find((result) => result.error)
  );
};

const handleCloseSingleResultView = (database: ComposedDatabase) => {
  tab.value?.queryContext?.results.delete(database.name || "");
};

// Auto select a proper database when the databases are ready.
watch(
  () => filteredItems.value,
  (items) => {
    const curr = props.selectedDatabase;
    if (!curr || !items.find((item) => item.database === curr)) {
      emit("update:selected-database", head(items)?.database);
    }
  },
  {
    immediate: true,
  }
);

const getBackgroundColorRgb = (database: ComposedDatabase) => {
  const color = hexToRgb(
    database.effectiveEnvironmentEntity.color || "#4f46e5"
  ).join(", ");
  return {
    backgroundColor: `rgba(${color}, 0.1)`,
    borderTopColor: `rgb(${color})`,
    color: `rgb(${color})`,
    borderTop: "3px solid",
  };
};
</script>
