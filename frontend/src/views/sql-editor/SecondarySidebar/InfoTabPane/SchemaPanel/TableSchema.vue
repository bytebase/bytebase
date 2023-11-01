<template>
  <div class="overflow-hidden flex flex-col">
    <div class="flex items-center justify-between pl-4 pr-2 py-1 border-b">
      <div
        class="flex items-center flex-1 truncate cursor-pointer"
        @click="emit('close')"
      >
        <heroicons-outline:table class="h-4 w-4 mr-1 flex-shrink-0" />
        <span v-if="schema.name" class="text-sm">{{ schema.name }}.</span>
        <span class="text-sm">{{ table.name }}</span>
      </div>

      <div v-if="pageMode === 'BUNDLED'" class="flex justify-end gap-x-0.5">
        <ExternalLinkButton
          :link="tableDetailLink"
          :tooltip="$t('common.detail')"
        />
        <AlterSchemaButton
          :database="database"
          :schema="schema"
          :table="table"
          @click="
            editorEvents.emit('alter-schema', {
              databaseUID: database.uid,
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
import { storeToRefs } from "pinia";
import { stringify } from "qs";
import { computed } from "vue";
import { useActuatorV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { databaseV1Slug } from "@/utils";
import { useSQLEditorContext } from "@/views/sql-editor/context";
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
}>();

const { events: editorEvents } = useSQLEditorContext();
const actuatorStore = useActuatorV1Store();
const { pageMode } = storeToRefs(actuatorStore);

const tableDetailLink = computed((): string => {
  const { database, schema, table } = props;
  const query: Record<string, string> = {
    table: table.name,
  };
  if (schema.name) {
    query.schema = schema.name;
  }
  const url = `/db/${databaseV1Slug(database)}?${stringify(query)}`;

  return url;
});
</script>
