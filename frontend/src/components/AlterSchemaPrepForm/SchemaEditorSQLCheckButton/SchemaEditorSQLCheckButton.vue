<template>
  <SQLCheckButton
    v-if="show"
    :database="database"
    :get-statement="getStatement"
    :change-type="
      useOnlineSchemaChange
        ? CheckRequest_ChangeType.DDL_GHOST
        : CheckRequest_ChangeType.DDL
    "
    :button-props="{
      size: 'small',
    }"
    :highlight-row-filter="
      (row) => row.checkResult.title === 'advice.online-migration'
    "
    class="justify-end"
  >
    <template #row-title-extra="{ row, confirm }">
      <NButton
        v-if="row.checkResult.title === 'advice.online-migration'"
        size="small"
        type="primary"
        @click="handleEnableOnlineChange(confirm)"
      >
        {{ $t("task.online-migration.enable") }}
      </NButton>
    </template>
  </SQLCheckButton>
</template>
<script lang="ts" setup>
import { toRef } from "vue";
import { SQLCheckButton } from "@/components/SQLCheck";
import type { ComposedDatabase } from "@/types";
import { CheckRequest_ChangeType } from "@/types/proto/v1/sql_service";
import type { Defer } from "@/utils";
import { useSchemaEditorSQLCheck } from "./useSchemaEditorSQLCheck";

const props = defineProps<{
  databaseList: ComposedDatabase[];
  getStatement: () => Promise<{ statement: string; errors: string[] }>;
  useOnlineSchemaChange: boolean;
}>();

const emit = defineEmits<{
  (event: "enable-online-schema-change"): void;
}>();

const { show, database } = useSchemaEditorSQLCheck({
  databaseList: toRef(props, "databaseList"),
});

const handleEnableOnlineChange = (confirm: Defer<boolean> | undefined) => {
  emit("enable-online-schema-change");
  confirm?.resolve(false);
};
</script>
