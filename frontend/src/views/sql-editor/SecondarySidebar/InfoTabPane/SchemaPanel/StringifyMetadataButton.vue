<template>
  <NPopover
    trigger="click"
    placement="bottom-end"
    @update:show="(show: boolean) => show"
  >
    <template #trigger>
      <NButton quaternary size="tiny" class="!px-1" v-bind="$attrs">
        <CodeIcon class="w-4 h-4" />
      </NButton>
    </template>
    <TableSchemaViewer
      class="!w-[32rem] !h-[20rem]"
      :database="database"
      :schema="schema?.name"
      :table="table?.name"
    />
  </NPopover>
</template>

<script lang="ts" setup>
import { CodeIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import TableSchemaViewer from "@/components/TableSchemaViewer.vue";
import type { ComposedDatabase } from "@/types";
import {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

defineProps<{
  database: ComposedDatabase;
  schema?: SchemaMetadata;
  table?: TableMetadata;
}>();
</script>
