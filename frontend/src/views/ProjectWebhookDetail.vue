<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="flex justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
        <template v-if="projectWebhook.type === Webhook_Type.TYPE_SLACK">
          <img class="h-6 w-6" src="../assets/slack-logo.png" alt="" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_DISCORD">
          <img class="h-6 w-6" src="../assets/discord-logo.svg" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_TEAMS">
          <img class="h-6 w-6" src="../assets/teams-logo.svg" />
        </template>
        <template
          v-else-if="projectWebhook.type === Webhook_Type.TYPE_DINGTALK"
        >
          <img class="h-6 w-6" src="../assets/dingtalk-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_FEISHU">
          <img class="h-6 w-6" src="../assets/feishu-logo.webp" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_WECOM">
          <img class="h-6 w-6" src="../assets/wecom-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_CUSTOM">
          <heroicons-outline:puzzle class="w-6 h-6" />
        </template>
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

import ProjectWebhookForm from "../components/ProjectWebhookForm.vue";
import { idFromSlug } from "../utils";
import { useProjectV1Store, useProjectWebhookV1Store } from "@/store";
import { Webhook_Type } from "@/types/proto/v1/project_service";
import { emptyProjectWebhook } from "@/types";

const props = defineProps({
  projectSlug: {
    required: true,
    type: String,
  },
  projectWebhookSlug: {
    required: true,
    type: String,
  },
  allowEdit: {
    required: true,
    type: Boolean,
  },
});

const projectV1Store = useProjectV1Store();
const projectWebhookV1Store = useProjectWebhookV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByUID(String(idFromSlug(props.projectSlug)));
});

const projectWebhook = computed(() => {
  const id = idFromSlug(props.projectWebhookSlug);
  return (
    projectWebhookV1Store.getProjectWebhookFromProjectById(project.value, id) ??
    emptyProjectWebhook()
  );
});
</script>
