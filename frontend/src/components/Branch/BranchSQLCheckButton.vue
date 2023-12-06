<template>
  <SQLCheckButton
    v-if="database"
    :database="database"
    :get-statement="getStatement"
    class="justify-end"
    :button-style="{
      height: '28px',
    }"
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
import { computed } from "vue";
import { SQLCheckButton, SQLCheckSummary } from "@/components/SQLCheck";
import { useDatabaseV1Store } from "@/store";
import { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  branch: Branch;
  getStatement: () => Promise<{ statement: string; errors: string[] }>;
}>();

const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(props.branch.baselineDatabase);
});
</script>
