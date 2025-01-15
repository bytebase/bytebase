<template>
  <div v-if="code === 1802" class="textinfolabel">
    {{
      $t(
        "task.online-migration.advice-split-statement-to-single-online-migration-issue"
      )
    }}
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { PlanCheckDetailTableRow } from "../PlanCheckRunDetail.vue";

const props = defineProps<{
  row: PlanCheckDetailTableRow;
}>();

const code = computed(() => {
  return props.row.checkResult.code;
});
</script>
