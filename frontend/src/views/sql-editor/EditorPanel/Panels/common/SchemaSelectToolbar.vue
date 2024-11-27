<template>
  <template v-if="showSchemaSelect">
    <NSelect
      v-if="simple"
      v-model:value="selectedSchemaName"
      :options="schemaSelectOptions"
      size="small"
      class="min-w-[8rem]"
      v-bind="$attrs"
      :consistent-menu-width="false"
    />

    <div v-else class="w-full flex flex-row justify-between items-center">
      <div class="flex flex-row justify-start items-center text-sm gap-x-2">
        <div class="flex items-center gap-x-1">
          <SchemaIcon class="w-4 h-4" />
          <span>{{ $t("common.schema") }}:</span>
        </div>
        <NSelect
          v-model:value="selectedSchemaName"
          :options="schemaSelectOptions"
          size="small"
          class="min-w-[8rem]"
          v-bind="$attrs"
        />
      </div>
    </div>
  </template>
</template>

<script setup lang="ts">
import { NSelect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SchemaIcon } from "@/components/Icon";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { hasSchemaProperty } from "@/utils";
import { useEditorPanelContext } from "../../context";

defineOptions({
  inheritAttrs: false,
});

defineProps<{
  simple?: boolean;
}>();

const { t } = useI18n();
const { database, instance } = useConnectionOfCurrentSQLEditorTab();
const { selectedSchemaName } = useEditorPanelContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});
const schemaSelectOptions = computed(() => {
  return databaseMetadata.value.schemas.map<SelectOption>((schema) => ({
    label: schema.name || t("db.schema.default"),
    value: schema.name,
  }));
});

const showSchemaSelect = computed(() => {
  return hasSchemaProperty(instance.value.engine);
});
</script>
