<template>
  <router-link
    v-if="issueUID"
    :to="`/issue/${issueUID}`"
    class="normal-link"
    target="_blank"
    @click.stop
    >#{{ issueUID }}</router-link
  >
  <span v-else>-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { extractIssueUID } from "@/utils";

const props = defineProps<{
  changeHistory: ChangeHistory;
}>();

const issueUID = computed(() => {
  return extractIssueUID(props.changeHistory.issue);
});
</script>
