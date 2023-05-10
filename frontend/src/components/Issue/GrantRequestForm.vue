<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-4 my-8"
  >
    <!-- TODO(steven): implement request grant form with IAM condition expr -->
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center">
        {{ $t("database.sync-schema.select-project") }}
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :disabled="!create"
        :selected-id="projectId"
        @select-project-id="handleSourceProjectSelect"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useIssueLogic } from "./logic";
import { Issue, IssueCreate, ProjectId } from "@/types";

const { create, issue } = useIssueLogic();

const projectId = computed(() => {
  return create.value
    ? (issue.value as IssueCreate).projectId
    : (issue.value as Issue).project.id;
});

const handleSourceProjectSelect = (projectId: ProjectId) => {
  if (!create.value) {
    return;
  }

  (issue.value as IssueCreate).projectId = projectId;
};
</script>
