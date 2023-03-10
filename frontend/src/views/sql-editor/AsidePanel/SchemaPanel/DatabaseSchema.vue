<template>
  <div v-if="databaseMetadata" class="h-full overflow-hidden flex flex-col">
    <div class="flex items-center justify-between p-2 border-b gap-x-1">
      <div
        class="flex items-center flex-1 truncate"
        :class="[headerClickable && 'cursor-pointer']"
        @click="handleClickHeader"
      >
        <heroicons-outline:database class="h-4 w-4 mr-1 flex-shrink-0" />
        <span class="font-semibold">{{ databaseMetadata.name }}</span>
      </div>
      <div class="flex justify-end gap-x-0.5">
        <SchemaDiagramButton
          v-if="instanceHasAlterSchema(database.instance)"
          :database="database"
          :database-metadata="databaseMetadata"
        />
        <ExternalLinkButton
          :link="`/db/${databaseSlug(database)}`"
          :tooltip="$t('common.detail')"
        />
        <AlterSchemaButton
          v-if="instanceHasAlterSchema(database.instance)"
          :database="database"
          @click="
            emit('alter-schema', {
              databaseId: database.id,
              schema: '',
              table: '',
            })
          "
        />
      </div>
    </div>

    <div class="flex-1 p-1 overflow-y-auto flex flex-col gap-y-2">
      <template v-for="(schema, i) in databaseMetadata.schemas" :key="i">
        <div v-for="(table, j) in schema.tables" :key="j" class="text-sm">
          <div
            class="flex items-center h-6 px-1 text-gray-600 whitespace-pre-wrap break-words rounded-sm"
            :class="
              rowClickable && ['hover:bg-[rgb(243,243,245)]', 'cursor-pointer']
            "
            @click="handleClickTable(schema, table)"
          >
            <heroicons-outline:table class="h-4 w-4 mr-1" />
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
import type { Database, DatabaseId } from "@/types";
import { databaseSlug, instanceHasAlterSchema } from "@/utils";
import ExternalLinkButton from "./ExternalLinkButton.vue";
import AlterSchemaButton from "./AlterSchemaButton.vue";
import SchemaDiagramButton from "./SchemaDiagramButton.vue";

const props = defineProps<{
  database: Database;
  databaseMetadata: DatabaseMetadata;
  headerClickable: boolean;
}>();

const emit = defineEmits<{
  (e: "click-header"): void;
  (e: "select-table", schema: SchemaMetadata, table: TableMetadata): void;
  (
    event: "alter-schema",
    params: { databaseId: DatabaseId; schema: string; table: string }
  ): void;
}>();

const engine = computed(() => props.database.instance.engine);

const rowClickable = computed(() => engine.value !== "MONGODB");

const handleClickHeader = () => {
  if (!props.headerClickable) return;
  emit("click-header");
};

const handleClickTable = (schema: SchemaMetadata, table: TableMetadata) => {
  if (!rowClickable.value) {
    return;
  }
  emit("select-table", schema, table);
};
</script>
