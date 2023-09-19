<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div v-if="allowEdit" class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click.prevent="addProjectWebhook"
      >
        {{ $t("project.webhook.add-a-webhook") }}
      </button>
    </div>
    <div class="pt-4">
      <div v-if="projectWebhookList.length > 0" class="space-y-6">
        <template
          v-for="(projectWebhook, index) in projectWebhookList"
          :key="index"
        >
          <ProjectWebhookCard :project-webhook="projectWebhook" />
        </template>
      </div>
      <template v-else>
        <div class="text-center">
          <heroicons-outline:inbox
            class="mx-auto w-16 h-16 text-control-light"
          />
          <h3 class="mt-2 text-sm font-medium text-main">
            {{ $t("project.webhook.no-webhook.title") }}
          </h3>
          <p class="mt-1 text-sm text-control-light">
            {{ $t("project.webhook.no-webhook.content") }}
          </p>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useRouter } from "vue-router";
import { Project } from "@/types/proto/v1/project_service";
import ProjectWebhookCard from "./ProjectWebhookCard.vue";

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});
const router = useRouter();

const projectWebhookList = computed(() => {
  return props.project.webhooks;
});

const addProjectWebhook = () => {
  router.push({
    name: "workspace.project.hook.create",
  });
};
</script>
