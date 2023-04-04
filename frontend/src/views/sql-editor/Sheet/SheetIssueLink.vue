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
import { isUndefined } from "lodash-es";
import { NTooltip } from "naive-ui";

import { Sheet, SheetIssueBacktracePayload } from "@/types";
import { issueSlug } from "@/utils";

const props = defineProps<{
  sheet: Sheet;
}>();

const payload = computed(() => {
  const maybePayload = (props.sheet.payload ??
    {}) as SheetIssueBacktracePayload;
  // if (typeof maybePayload.)
  if (
    !isUndefined(maybePayload.issueId) &&
    !isUndefined(maybePayload.issueName)
  ) {
    return maybePayload;
  }

  return undefined;
});
</script>
