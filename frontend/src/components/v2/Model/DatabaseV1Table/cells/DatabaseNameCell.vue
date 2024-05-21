<template>
  <div class="flex items-center space-x-2">
    <SQLEditorButtonV1
      v-if="showSQLEditorButton"
      :database="database"
      :disabled="!allowQuery"
      :tooltip="true"
      @failed="handleGotoSQLEditorFailed"
    />
    <DatabaseV1Name :database="database" :link="false" tag="span" />
  </div>
</template>

<script setup lang="ts">
import SQLEditorButtonV1 from "@/components/DatabaseDetail/SQLEditorButtonV1.vue";
import { DatabaseV1Name } from "@/components/v2";
import { pushNotification, useCurrentUserV1 } from "@/store";
import type { ComposedDatabase } from "@/types";
import { isDatabaseV1Queryable } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
  showSQLEditorButton: boolean;
}>();

const currentUser = useCurrentUserV1();

const allowQuery = () => {
  return isDatabaseV1Queryable(props.database, currentUser.value);
};

const handleGotoSQLEditorFailed = () => {
  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: `Failed to go to SQL editor for database ${props.database.name}`,
  });
};
</script>
