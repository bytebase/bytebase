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
import { toRefs } from "vue";
import { SQLCheckButton, SQLCheckSummary } from "@/components/SQLCheck";
import { ComposedDatabase } from "@/types";
import { useSchemaEditorSQLCheck } from "./useSchemaEditorSQLCheck";

const props = defineProps<{
  selectedTab: "raw-sql" | "schema-editor";
  databaseList: ComposedDatabase[];
  editStatement: string;
}>();

const { show, database, getStatement } = useSchemaEditorSQLCheck(toRefs(props));
</script>
