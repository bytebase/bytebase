<template>
  <SQLCheckButton
    v-if="show"
    :database="database"
    :get-statement="getStatement"
    :change-type="Release_File_ChangeType.DDL"
    :button-props="{
      size: 'small',
    }"
    :show-code-location="false"
    :advice-filter="(advice) => advice.title !== 'advice.online-migration'"
    class="justify-end"
  />
</template>
<script lang="ts" setup>
import { toRef } from "vue";
import { SQLCheckButton } from "@/components/SQLCheck";
import type { ComposedDatabase } from "@/types";
import { Release_File_ChangeType } from "@/types/proto-es/v1/release_service_pb";
import { useSchemaEditorSQLCheck } from "./useSchemaEditorSQLCheck";

const props = defineProps<{
  databaseList: ComposedDatabase[];
  getStatement: () => Promise<{
    statement: string;
    errors: string[];
  }>;
}>();

const { show, database } = useSchemaEditorSQLCheck({
  databaseList: toRef(props, "databaseList"),
});
</script>
