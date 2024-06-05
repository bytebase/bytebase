<template>
  <SQLCheckButton
    v-if="show"
    :database="database"
    :get-statement="getStatement"
    :button-props="{
      size: 'small',
    }"
    class="justify-end"
  />
</template>
<script lang="ts" setup>
import { toRef } from "vue";
import { SQLCheckButton } from "@/components/SQLCheck";
import type { ComposedDatabase } from "@/types";
import { useSchemaEditorSQLCheck } from "./useSchemaEditorSQLCheck";

const props = defineProps<{
  databaseList: ComposedDatabase[];
  getStatement: () => Promise<{ statement: string; errors: string[] }>;
}>();

const { show, database } = useSchemaEditorSQLCheck({
  databaseList: toRef(props, "databaseList"),
});
</script>
