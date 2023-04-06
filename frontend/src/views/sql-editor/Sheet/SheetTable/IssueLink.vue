<template>
  <NTooltip v-if="payload">
    <template #trigger>
      <a
        class="normal-link"
        target="_blank"
        :href="`/issue/${issueSlug(payload.issueName, payload.issueId)}`"
        @click.stop=""
        >#{{ payload.issueId }}</a
      >
    </template>

    <div class="max-w-[20rem]">
      {{ payload.issueName }}
    </div>
  </NTooltip>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NTooltip } from "naive-ui";

import { Sheet } from "@/types";
import { getSheetIssueBacktracePayload, issueSlug } from "@/utils";

const props = defineProps<{
  sheet: Sheet;
}>();

const payload = computed(() => {
  return getSheetIssueBacktracePayload(props.sheet);
});
</script>
