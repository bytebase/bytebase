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

const props = defineProps<{
  projectId: string;
  webhookResourceId: string;
  allowEdit: boolean;
}>();

const projectWebhookV1Store = useProjectWebhookV1Store();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const projectWebhook = computed(() => {
  return (
    projectWebhookV1Store.getProjectWebhookFromProjectById(
      project.value,
      props.webhookResourceId
    ) ?? unknownProjectWebhook()
  );
});
</script>
