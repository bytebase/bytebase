<template>
  <ProjectWebhookForm
    class="pt-4"
    :allow-edit="allowEdit"
    :create="false"
    :project="project"
    :webhook="projectWebhook"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProjectWebhookForm from "@/components/ProjectWebhookForm.vue";
import { useProjectByName, useProjectWebhookV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { unknownProjectWebhook } from "@/types/v1/projectWebhook";
import { idFromSlug } from "@/utils";

const props = defineProps<{
  projectId: string;
  projectWebhookSlug: string;
  allowEdit: boolean;
}>();

const projectWebhookV1Store = useProjectWebhookV1Store();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const projectWebhook = computed(() => {
  const id = idFromSlug(props.projectWebhookSlug);
  return (
    projectWebhookV1Store.getProjectWebhookFromProjectById(project.value, id) ??
    unknownProjectWebhook()
  );
});
</script>
