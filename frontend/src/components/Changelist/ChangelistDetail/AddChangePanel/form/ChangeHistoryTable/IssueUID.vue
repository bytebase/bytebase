<template>
  <div
    v-if="changeHistory.issueEntity"
    class="flex items-center overflow-hidden space-x-1"
  >
    <router-link
      :to="{
        path: `/${changeHistory.issueEntity.name}`,
      }"
      class="normal-link"
      target="_blank"
      @click.stop
    >
      #{{ extractIssueUID(changeHistory.issueEntity.name) }}
    </router-link>
    <NPerformantEllipsis class="flex-1 truncate">
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span class="textinfo" v-html="issueTitle" />
    </NPerformantEllipsis>
  </div>
  <span v-else>-</span>
</template>

<script setup lang="ts">
import { escape } from "lodash-es";
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { type ComposedChangeHistory } from "@/types";
import { extractIssueUID, getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  changeHistory: ComposedChangeHistory;
  keyword: string;
}>();

const issueTitle = computed(() => {
  const keyword = props.keyword.trim().toLowerCase();
  const title = props.changeHistory.issueEntity?.title ?? "";

  if (!keyword) {
    return title;
  }

  return getHighlightHTMLByRegExp(
    title,
    escape(keyword),
    false /* !caseSensitive */
  );
});
</script>
