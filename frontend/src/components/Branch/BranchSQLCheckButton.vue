<template>
  <SQLCheckButton
    v-if="database"
    :database="database"
    :database-metadata="databaseMetadata"
    :get-statement="getStatement"
    :button-style="{
      height: '28px',
    }"
    :advice-filter="(advice) => advice.title !== 'advice.online-migration'"
    class="justify-end"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SQLCheckButton } from "@/components/SQLCheck";
import { useDatabaseV1Store } from "@/store";
import type { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  branch: Branch;
  getStatement: () => Promise<{ statement: string; errors: string[] }>;
}>();

const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(props.branch.baselineDatabase);
});

const databaseMetadata = computed(() => {
  return props.branch.baselineSchemaMetadata;
});
</script>
