<template>
  <div class="overflow-hidden flex flex-col">
    <div class="flex items-center justify-between p-2 pl-4 border-b">
      <div
        class="flex items-center flex-1 truncate cursor-pointer"
        @click="emit('close')"
      >
        <heroicons-outline:table class="h-4 w-4 mr-1 flex-shrink-0" />
        <span v-if="schema.name" class="font-semibold">{{ schema.name }}.</span>
        <span class="font-semibold">{{ table.name }}</span>
      </div>

      <div
        v-if="sqlEditorStore.mode === 'BUNDLED'"
        class="flex justify-end gap-x-0.5"
      >
        <ExternalLinkButton
          :link="tableDetailLink"
          :tooltip="$t('common.detail')"
        />
        <AlterSchemaButton
          :database="database"
          :schema="schema"
          :table="table"
          @click="
            emit('alter-schema', {
              databaseId: database.uid,
              schema: schema.name,
              table: table.name,
            })
          "
        />
      </div>
    </div>

    <ColumnList :table="table" class="w-full flex-1 py-1" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useSQLEditorStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { databaseV1Slug } from "@/utils";
import AlterSchemaButton from "./AlterSchemaButton.vue";
import ColumnList from "./ColumnList.vue";
import ExternalLinkButton from "./ExternalLinkButton.vue";

const props = defineProps<{
  database: ComposedDatabase;
  databaseMetadata: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();

const sqlEditorStore = useSQLEditorStore();

const tableDetailLink = computed((): string => {
  const { database, schema, table } = props;
  let url = `/db/${databaseV1Slug(database)}/table/${encodeURIComponent(
    table.name
  )}`;
  if (schema.name) {
    url += `?schema=${encodeURIComponent(schema.name)}`;
  }

  return url;
});
</script>
