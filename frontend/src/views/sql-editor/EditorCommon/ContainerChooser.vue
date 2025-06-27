<template>
  <ConnectChooser
    v-if="show"
    v-model:value="chosenContainer"
    :options="options"
    :is-chosen="isChosen"
    :placeholder="$t('database.table.select')"
  />
</template>

<script lang="tsx" setup>
import { type SelectOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import ConnectChooser from "./ConnectChooser.vue";

const OptionValueUnspecified = "-1";

const { t } = useI18n();
const route = useRoute();
const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { database, instance } = useConnectionOfCurrentSQLEditorTab();
const show = computed(() => {
  return instance.value.engine === Engine.COSMOSDB;
});

const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(database.value.name);
});
const options = computed(() => {
  const selectOptions: SelectOption[] = [
    {
      value: OptionValueUnspecified,
      label: t("database.schema.unspecified"),
    },
  ];

  for (const schema of databaseMetadata.value.schemas) {
    for (const table of schema.tables) {
      selectOptions.push({
        value: table.name,
        label: table.name,
      });
    }
  }
  return selectOptions;
});

const chosenContainer = computed<string>({
  get() {
    const table = tab.value?.connection.table;
    if (table === undefined) return OptionValueUnspecified;
    return table;
  },
  set(value) {
    if (!tab.value) return;
    tab.value.connection.table =
      value === OptionValueUnspecified ? undefined : value;
  },
});

watchEffect(() => {
  if (route.query.table) {
    chosenContainer.value = route.query.table as string;
  }
});

const isChosen = computed(() => {
  return chosenContainer.value !== OptionValueUnspecified;
});
</script>
