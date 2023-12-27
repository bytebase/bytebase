<template>
  <div class="w-full pt-2 px-2">
    <NAutoComplete
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
    >
      <template #prefix>
        <SearchIcon class="w-4 h-auto text-gray-300" />
      </template>
    </NAutoComplete>
  </div>
</template>

<script lang="ts" setup>
import { SearchIcon } from "lucide-vue-next";
import { NAutoComplete, SelectOption } from "naive-ui";
import { computed, watchEffect, h } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { CoreTabInfo, TabMode } from "@/types";
import { emptyConnection, tryConnectToCoreTab } from "@/utils";
import useSearchHistory from "./useSearchHistory";

const props = defineProps<{
  searchPattern: string;
}>();

defineEmits<{
  (event: "update:search-pattern", keyword: string): void;
}>();

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const { searchHistory } = useSearchHistory();

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
    return h("div", { class: "w-full text-sm text-gray-400" }, [option.label]);
  }

  const database = databaseStore.getDatabaseByName(option.value as string);
  return h(
    "div",
    {
      class: "w-full flex items-center gap-x-1",
    },
    [
      h(InstanceV1EngineIcon, {
        instance: database.instanceEntity,
      }),
      h(EnvironmentV1Name, {
        environment: database.effectiveEnvironmentEntity,
        link: false,
        class: "text-control-light",
      }),
      h("span", {}, database.databaseName),
    ]
  );
};

const handleDatabaseSelect = (databaseName: string) => {
  const database = databaseStore.getDatabaseByName(databaseName);
  const coreTab: CoreTabInfo = {
    connection: {
      ...emptyConnection(),
      instanceId: database.instanceEntity.uid,
      databaseId: database.uid,
    },
    sheetName: undefined,
    mode: TabMode.ReadOnly,
  };
  tryConnectToCoreTab(coreTab);
};

watchEffect(async () => {
  for (const databaseName of searchHistory.value) {
    const database = await databaseStore.getOrFetchDatabaseByName(
      databaseName,
      true /* silent */
    );
    if (!database) {
      continue;
    }
  }
});
</script>
