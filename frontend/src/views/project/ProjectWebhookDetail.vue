<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="flex justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <WebhookTypeIcon :type="projectWebhook.type" class="h-6 w-6" />
        <h3 class="text-xl leading-6 font-medium text-main">
          {{ projectWebhook.title }}
        </h3>
      </div>
    </div>
    <ProjectWebhookForm
      class="pt-4"
      :allow-edit="allowEdit"
      :create="false"
      :project="project"
      :webhook="projectWebhook"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProjectWebhookForm from "@/components/ProjectWebhookForm.vue";
import { useProjectV1Store, useProjectWebhookV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { emptyProjectWebhook } from "@/types";
import { idFromSlug } from "@/utils";

const props = defineProps<{
  projectId: string;
  projectWebhookSlug: string;
  allowEdit: boolean;
}>();

const projectV1Store = useProjectV1Store();
const projectWebhookV1Store = useProjectWebhookV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const projectWebhook = computed(() => {
  const id = idFromSlug(props.projectWebhookSlug);
  return (
    projectWebhookV1Store.getProjectWebhookFromProjectById(project.value, id) ??
    emptyProjectWebhook()
  );
});
</script>
