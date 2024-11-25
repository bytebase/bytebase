<template>
  <NAutoComplete
    ref="autoCompleteRef"
    size="small"
    clear-after-select
    blur-after-select
    :clearable="true"
    :value="searchPattern"
    :placeholder="$t('sql-editor.search-databases')"
    :options="autoCompleteOptions"
    :render-label="renderLabel"
    :get-show="getOptionShow"
    @update:value="$emit('update:search-pattern', $event || '')"
    @select="handleDatabaseSelect"
    @compositionstart="isIMECompositing = true"
    @compositionend="isIMECompositing = false"
    @keydown.esc="handleEscapeKey"
  >
    <template #prefix>
      <SearchIcon class="w-4 h-auto text-gray-300" />
    </template>
  </NAutoComplete>
</template>

<script lang="ts" setup>
import { SearchIcon } from "lucide-vue-next";
import type { AutoCompleteInst, SelectOption } from "naive-ui";
import { NAutoComplete } from "naive-ui";
import { computed, watchEffect, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import type { CoreSQLEditorTab } from "@/types";
import { DEFAULT_SQL_EDITOR_TAB_MODE, isValidDatabaseName } from "@/types";
import {
  emptySQLEditorConnection,
  tryConnectToCoreSQLEditorTab,
} from "@/utils";
import useSearchHistory from "./useSearchHistory";

defineOptions({
  name: "SearchBox",
});

const props = defineProps<{
  searchPattern: string;
}>();

defineEmits<{
  (event: "update:search-pattern", keyword: string): void;
}>();

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const { searchHistory } = useSearchHistory();
const isIMECompositing = ref(false);
const autoCompleteRef = ref<AutoCompleteInst>();

const getOptionShow = () => {
  if (autoCompleteOptions.value[0].children.length === 0) {
    return false;
  }
  return true;
};

const autoCompleteOptions = computed(() => {
  const databaseNames = searchHistory.value;
  return [
    {
      type: "group",
      key: "recent",
      label: t("common.recent"),
      children: databaseNames
        .filter((databaseName) => {
          const database = databaseStore.getDatabaseByName(databaseName);
          return (
            database &&
            isValidDatabaseName(database.name) &&
            database.name
              .toLowerCase()
              .includes((props.searchPattern || "").toLowerCase())
          );
        })
        .slice(0, 5) // Only show 5 recent databases.
        .map<SelectOption>((databaseName) => {
          return {
            label: databaseName,
            value: databaseName,
          };
        }),
    },
  ];
});

const renderLabel = (option: SelectOption) => {
  if (option.type === "group") {
    return h(
      "div",
      { class: "w-full text-sm text-gray-400" },
      String(option.label)
    );
  }

  const database = databaseStore.getDatabaseByName(option.value as string);
  return h(
    "div",
    {
      class: "w-full flex items-center gap-x-1",
    },
    [
      h(InstanceV1EngineIcon, {
        instance: database.instanceResource,
      }),
      h(EnvironmentV1Name, {
        environment: database.effectiveEnvironmentEntity,
        link: false,
        class: "text-control-light",
      }),
      h(
        "span",
        {
          class: "truncate",
        },
        database.databaseName
      ),
    ]
  );
};

const handleDatabaseSelect = (databaseName: string) => {
  const database = databaseStore.getDatabaseByName(databaseName);
  const coreTab: CoreSQLEditorTab = {
    connection: {
      ...emptySQLEditorConnection(),
      instance: database.instance,
      database: database.name,
    },
    worksheet: "",
    mode: DEFAULT_SQL_EDITOR_TAB_MODE,
  };
  tryConnectToCoreSQLEditorTab(coreTab);
};

const handleEscapeKey = (_e: KeyboardEvent) => {
  if (isIMECompositing.value) return;
  autoCompleteRef.value?.blur();
};

watchEffect(async () => {
  for (const databaseName of searchHistory.value) {
    try {
      await databaseStore.getOrFetchDatabaseByName(
        databaseName,
        true /* silent */
      );
    } catch {
      // nothing
    }
  }
});
</script>
