<template>
  <NPopover trigger="click" placement="bottom-end">
    <template #trigger>
      <NButton
        quaternary
        size="tiny"
        class="!px-1"
        v-bind="$attrs"
        @click="handleShowSchemaString"
      >
        <Code class="w-4 h-4" />
      </NButton>
    </template>
    <div class="w-[32rem] h-[20rem] flex flex-col justify-center items-center">
      <BBSpin v-if="isFetching" />
      <MonacoEditor
        v-else
        :value="schemaString || ''"
        :readonly="true"
        class="border w-full h-full"
      />
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { Code } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { onMounted, ref } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

const props = defineProps<{
  database: ComposedDatabase;
  schema?: SchemaMetadata;
  table?: TableMetadata;
}>();

const dbSchemaStore = useDBSchemaV1Store();
const schemaString = ref<string | null>(null);
const isFetching = ref<boolean>(false);

onMounted(async () => {
  await dbSchemaStore.getOrFetchDatabaseMetadata({
    database: props.database.name,
    silent: true,
  });
});

const handleShowSchemaString = async () => {
  const databaseMetadata = DatabaseMetadata.fromPartial({
    name: props.database.name,
  });
  if (props.schema) {
    const schemaMetadata = SchemaMetadata.fromPartial({
      name: props.schema.name,
    });
    if (props.table) {
      schemaMetadata.tables = [
        TableMetadata.fromPartial({
          ...cloneDeep(props.table),
          foreignKeys: [],
        }),
      ];
    }
    databaseMetadata.schemas = [schemaMetadata];
  }

  isFetching.value = true;
  const { schema } = await sqlServiceClient.stringifyMetadata({
    metadata: databaseMetadata,
    engine: props.database.instanceEntity.engine,
  });
  schemaString.value = schema.trim();
  isFetching.value = false;
};
</script>
