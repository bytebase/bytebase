<template>
  <ProjectMemberPanel class="py-4" :project="project" :allow-edit="allowEdit" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectMemberPanel from "@/components/ProjectMember/ProjectMemberPanel.vue";
import { useActuatorV1Store, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { isDefaultProject } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const actuatorStore = useActuatorV1Store();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const allowEdit = computed(() => {
  if (isDefaultProject(project.value.name, actuatorStore.serverInfo?.workspace ?? "")) {
    return false;
  }

  if (project.value.state === State.DELETED) {
    return false;
  }

  return hasProjectPermissionV2(project.value, "bb.projects.setIamPolicy");
});
</script>
