<template>
  <SlowQueryPanel
    :readonly-search-scopes="readonlyScopes"
    :support-option-id-list="['environment', 'instance', 'database']"
    :show-project-column="false"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SlowQueryPanel } from "@/components/SlowQuery";
import type { Project } from "@/types/proto/v1/project_service";
import type { SearchScope } from "@/utils";
import { extractProjectResourceName } from "@/utils";

const props = defineProps<{
  project: Project;
}>();

const readonlyScopes = computed((): SearchScope[] => {
  return [
    {
      id: "project",
      value: extractProjectResourceName(props.project.name),
    },
  ];
});
</script>
