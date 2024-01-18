<template>
  <ProjectMemberPanel :project="project" :allow-edit="allowEdit" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useProjectV1Store, useCurrentUserV1 } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const allowEdit = computed(() => {
  if (project.value.name === DEFAULT_PROJECT_V1_NAME) {
    return false;
  }

  if (project.value.state === State.DELETED) {
    return false;
  }

  return hasProjectPermissionV2(
    project.value,
    useCurrentUserV1().value,
    "bb.projects.setIamPolicy"
  );
});
</script>
