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
          v-if="instanceV1HasAlterSchema(database.instanceEntity)"
          :database="database"
          :database-metadata="databaseMetadata"
        />
        <ExternalLinkButton
          :link="`/db/${databaseV1Slug(database)}`"
          :tooltip="$t('common.detail')"
        />
        <AlterSchemaButton
          v-if="instanceV1HasAlterSchema(database.instanceEntity)"
          :database="database"
          @click="
            emit('alter-schema', {
              databaseId: database.uid,
              schema: '',
              table: '',
            })
          "
        />
      </div>
    </div>

    <TableList
      class="flex-1 w-full py-1"
      :schema-list="availableSchemas"
      :row-clickable="rowClickable"
      @select-table="(schema, table) => $emit('select-table', schema, table)"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import {
  databaseV1Slug,
  instanceV1HasAlterSchema,
  isTableQueryable,
} from "@/utils";
import AlterSchemaButton from "./AlterSchemaButton.vue";
import ExternalLinkButton from "./ExternalLinkButton.vue";
import SchemaDiagramButton from "./SchemaDiagramButton.vue";
import TableList from "./TableList.vue";

const props = defineProps<{
  database: ComposedDatabase;
  databaseMetadata: DatabaseMetadata;
  headerClickable: boolean;
}>();

const emit = defineEmits<{
  (e: "click-header"): void;
  (e: "select-table", schema: SchemaMetadata, table: TableMetadata): void;
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();

const currentUser = useCurrentUserV1();

const engine = computed(() => props.database.instanceEntity.engine);

const rowClickable = computed(() => engine.value !== Engine.MONGODB);

const availableSchemas = computed(() => {
  const schemas = props.databaseMetadata.schemas
    .map((schema) => {
      return {
        ...schema,
        tables: schema.tables.filter((table) => {
          return isTableQueryable(
            props.database,
            schema.name,
            table.name,
            currentUser.value
          );
        }),
      };
    })
    .filter((schema) => {
      return schema.tables.length !== 0;
    });
  return schemas;
});

const handleClickHeader = () => {
  if (!props.headerClickable) return;
  emit("click-header");
};
</script>
