<template>
  <div class="h-full flex flex-col">
    <div class="text-lg leading-6 font-medium text-main">
      {{ $t("project.webhook.creation.title") }}
    </div>
    <NDivider />
    <ProjectWebhookForm
      class="flex-1"
      :create="true"
      :project="project"
      :webhook="defaultNewWebhook"
    />
  </div>
</template>

<script lang="ts" setup>
import { NDivider } from "naive-ui";
import { computed } from "vue";
import ProjectWebhookForm from "@/components/ProjectWebhookForm.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { emptyProjectWebhook } from "@/types";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const defaultNewWebhook = emptyProjectWebhook();
</script>
