<template>
  <ProjectMemberPanel :project="project" :allow-edit="allowEdit" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectMemberPanel from "@/components/ProjectMember/ProjectMemberPanel.vue";
import { useCurrentUserV1, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { DEFAULT_PROJECT_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const allowEdit = computed(() => {
  if (project.value.name === DEFAULT_PROJECT_NAME) {
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
