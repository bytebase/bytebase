<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div v-if="allowEdit" class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click.prevent="addProjectWebhook"
      >
        Add a webhook
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
          <svg
            class="mx-auto w-16 h-16 text-control-light"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
            ></path>
          </svg>
          <h3 class="mt-2 text-sm font-medium text-main">
            No webhook configured for this project.
          </h3>
          <p class="mt-1 text-sm text-control-light">
            Configure webhooks to let Bytebase post notification to the external
            systems on various events.
          </p>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, PropType, watchEffect } from "@vue/runtime-core";
import { useRouter } from "vue-router";
import ProjectWebhookCard from "./ProjectWebhookCard.vue";
import { Project } from "../types";
import { useStore } from "vuex";

export default {
  name: "ProjectWebhookPanel",
  components: {
    ProjectWebhookCard,
  },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    allowEdit: {
      default: true,
      type: Boolean,
    },
  },
  setup(props) {
    const store = useStore();
    const router = useRouter();

    const prepareProjectWebhookList = () => {
      store.dispatch(
        "projectWebhook/fetchProjectWebhookListByProjectId",
        props.project.id
      );
    };

    watchEffect(prepareProjectWebhookList);

    const projectWebhookList = computed(() => {
      return store.getters["projectWebhook/projectWebhookListByProjectId"](
        props.project.id
      );
    });

    const addProjectWebhook = () => {
      router.push({
        name: "workspace.project.hook.create",
      });
    };

    return {
      projectWebhookList,
      addProjectWebhook,
    };
  },
};
</script>
