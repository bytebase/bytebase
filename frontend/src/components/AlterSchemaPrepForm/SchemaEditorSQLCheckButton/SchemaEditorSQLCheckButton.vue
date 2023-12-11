<template>
  <SQLCheckButton
    v-if="show"
    :database="database"
    :get-statement="getStatement"
    :button-props="{
      size: 'small',
    }"
    class="justify-end"
  >
    <template #result="{ advices, isRunning }">
      <SQLCheckSummary
        v-if="advices !== undefined && !isRunning"
        :database="database"
        :advices="advices"
      />
    </template>
  </SQLCheckButton>
</template>
<script lang="ts" setup>
import { toRef } from "vue";
import { SQLCheckButton, SQLCheckSummary } from "@/components/SQLCheck";
import { ComposedDatabase } from "@/types";
import { useSchemaEditorSQLCheck } from "./useSchemaEditorSQLCheck";

const props = defineProps<{
  databaseList: ComposedDatabase[];
  getStatement: () => Promise<{ statement: string; errors: string[] }>;
}>();

const { show, database } = useSchemaEditorSQLCheck({
  databaseList: toRef(props, "databaseList"),
});
</script>
