<template>
  <ProjectSettingPanel :project="project" :allow-edit="allowEdit" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectSettingPanel from "@/components/ProjectSettingPanel.vue";
import { useBodyLayoutContext } from "@/layouts/common";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("py-0!");
</script>
