<template>
  <router-link
    v-if="project && issueUID"
    :to="{
      name: 'workspace.project.issue.detail',
      params: {
        projectId: project,
        issueSlug: issueUID,
      },
    }"
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
import { extractIssueUID, extractProjectResourceName } from "@/utils";

const props = defineProps<{
  changeHistory: ChangeHistory;
}>();

const project = computed(() => {
  return extractProjectResourceName(props.changeHistory.issue);
});
const issueUID = computed(() => {
  return extractIssueUID(props.changeHistory.issue);
});
</script>
