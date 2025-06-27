<template>
  <ConnectChooser
    v-if="show"
    v-model:value="chosenSchema"
    :options="options"
    :is-chosen="isChosen"
    :placeholder="$t('database.schema.select')"
  />
</template>

<script setup lang="ts">
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
import { instanceAllowsSchemaScopedQuery } from "@/utils";
import { convertEngineToNew } from "@/utils/v1/common-conversions";
import ConnectChooser from "./ConnectChooser.vue";

const SchemaOptionValueUnspecified = "-1";

const { t } = useI18n();
const route = useRoute();
const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { database, instance } = useConnectionOfCurrentSQLEditorTab();
const show = computed(() => {
  return instanceAllowsSchemaScopedQuery(convertEngineToNew(instance.value.engine));
});

const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(database.value.name);
});
const options = computed(() => {
  const options = databaseMetadata.value.schemas.map<SelectOption>(
    (schema) => ({
      value: schema.name,
      label: schema.name || t("db.schema.default"),
    })
  );
  options.unshift({
    value: SchemaOptionValueUnspecified,
    label: t("database.schema.unspecified"),
  });
  return options;
});

const chosenSchema = computed<string>({
  get() {
    const schema = tab.value?.connection.schema;
    if (schema === undefined) return SchemaOptionValueUnspecified;
    return schema;
  },
  set(value) {
    if (!tab.value) return;
    tab.value.connection.schema =
      value === SchemaOptionValueUnspecified ? undefined : value;
  },
});

watchEffect(() => {
  if (route.query.schema) {
    chosenSchema.value = route.query.schema as string;
  }
});

const isChosen = computed(() => {
  return chosenSchema.value !== SchemaOptionValueUnspecified;
});
</script>
