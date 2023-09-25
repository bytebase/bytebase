<template>
  <SQLCheckButton
    v-if="show"
    :database="database"
    :statement="statement"
    :errors="errors"
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
import { debounce } from "lodash-es";
import { ref, toRefs, watch } from "vue";
import { SQLCheckButton, SQLCheckSummary } from "@/components/SQLCheck";
import { ComposedDatabase } from "@/types";
import { useSchemaEditorSQLCheck } from "./useSchemaEditorSQLCheck";

const props = defineProps<{
  selectedTab: "raw-sql" | "schema-editor";
  databaseList: ComposedDatabase[];
  editStatement: string;
}>();

const { show, database, watchKey, getStatement } = useSchemaEditorSQLCheck(
  toRefs(props)
);
const errors = ref<string[]>([]);
const statement = ref("");

const update = async () => {
  const result = await getStatement();
  errors.value = result.errors;
  statement.value = result.statement;
};

watch(watchKey, debounce(update, 250));
update();
</script>
