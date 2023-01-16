<template>
  <div v-if="databaseMetadata" class="h-full overflow-hidden flex flex-col">
    <div class="flex items-center justify-between p-2 border-b">
      <div class="flex items-center">
        <heroicons-outline:database class="h-4 w-4 mr-1" />
        <span class="font-semibold">{{ databaseMetadata.name }}</span>
      </div>
      <div class="flex justify-end space-x-2">
        <SchemaDiagramButton
          :database="database"
          :database-metadata="databaseMetadata"
        />
        <AlterSchemaButton :database="database" />
      </div>
    </div>
    <div class="px-2 py-2 border-b text-gray-500 text-xs space-y-1">
      <div v-if="showCharset" class="flex items-center justify-between">
        <span>
          {{
            engine == "POSTGRES" ? $t("db.encoding") : $t("db.character-set")
          }}
        </span>
        <span>{{ database.characterSet }}</span>
      </div>
      <div v-if="showCollation" class="flex items-center justify-between">
        <span>
          {{ $t("db.collation") }}
        </span>
        <span>{{ database.collation }}</span>
      </div>
    </div>
    <div class="flex-1 px-2 py-1 overflow-y-auto">
      <div class="text-sm text-gray-500 py-1">
        {{
          database.instance.engine !== "MONGODB"
            ? $t("db.tables")
            : $t("db.collections")
        }}
      </div>

      <template v-for="(schema, i) in databaseMetadata.schemas" :key="i">
        <div v-for="(table, j) in schema.tables" :key="j" class="text-xs py-1">
          <div
            class="inline-block text-gray-600"
            :class="
              rowClickable && [
                'hover:text-[var(--color-accent)]',
                'cursor-pointer',
              ]
            "
            @click="handleClickTable(schema, table)"
          >
            <span v-if="schema.name">{{ schema.name }}.</span>
            <span>{{ table.name }}</span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import type { Database } from "@/types";
import AlterSchemaButton from "./AlterSchemaButton.vue";
import SchemaDiagramButton from "./SchemaDiagramButton.vue";

const props = defineProps<{
  database: Database;
  databaseMetadata: DatabaseMetadata;
}>();

const emit = defineEmits<{
  (e: "select-table", schema: SchemaMetadata, table: TableMetadata): void;
}>();

const engine = computed(() => props.database.instance.engine);

const showCharset = computed(
  () =>
    engine.value !== "CLICKHOUSE" &&
    engine.value !== "SNOWFLAKE" &&
    engine.value !== "MONGODB"
);
const showCollation = computed(
  () =>
    engine.value !== "CLICKHOUSE" &&
    engine.value !== "SNOWFLAKE" &&
    engine.value !== "MONGODB"
);

const rowClickable = computed(() => engine.value !== "MONGODB");

const handleClickTable = (schema: SchemaMetadata, table: TableMetadata) => {
  if (!rowClickable.value) {
    return;
  }
  emit("select-table", schema, table);
};
</script>
