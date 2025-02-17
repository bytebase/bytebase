<template>
  <SQLCheckButton
    v-if="show"
    :database="database"
    :get-statement="getStatement"
    :change-type="
      useOnlineSchemaMigration
        ? Release_File_ChangeType.DDL_GHOST
        : Release_File_ChangeType.DDL
    "
    :button-props="{
      size: 'small',
    }"
    :show-code-location="false"
    :advice-filter="(advice) => advice.title !== 'advice.online-migration'"
    class="justify-end"
  >
    <template #row-title-extra="{ row, confirm }">
      <OnlineMigrationAdviceExtra
        v-if="row.checkResult.title === 'advice.online-migration'"
        :row="row"
        @toggle="handleToggleOnlineMigration($event, confirm)"
      />
    </template>
  </SQLCheckButton>
</template>
<script lang="ts" setup>
import { toRef } from "vue";
import { SQLCheckButton } from "@/components/SQLCheck";
import type { ComposedDatabase } from "@/types";
import { Release_File_ChangeType } from "@/types/proto/v1/release_service";
import type { Defer } from "@/utils";
import OnlineMigrationAdviceExtra from "./OnlineMigrationAdviceExtra.vue";
import { useSchemaEditorSQLCheck } from "./useSchemaEditorSQLCheck";

const props = defineProps<{
  databaseList: ComposedDatabase[];
  getStatement: () => Promise<{
    statement: string;
    errors: string[];
  }>;
  useOnlineSchemaMigration: boolean;
}>();

const emit = defineEmits<{
  (event: "toggle-online-schema-migration", on: boolean): void;
}>();

const { show, database } = useSchemaEditorSQLCheck({
  databaseList: toRef(props, "databaseList"),
});

const handleToggleOnlineMigration = (
  on: boolean,
  confirm: Defer<boolean> | undefined
) => {
  emit("toggle-online-schema-migration", on);
  confirm?.resolve(false);
};
</script>
