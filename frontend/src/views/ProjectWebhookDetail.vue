<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="flex justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <img class="h-6 w-6" :src="`/src/assets/${logo}`" />
        <h3 class="text-xl leading-6 font-medium text-main">
          {{ projectWebhook.name }}
        </h3>
      </div>
      <button
        @click.prevent="testWebhook"
        type="button"
        class="btn-normal whitespace-nowrap items-center"
      >
        Test Webhook
      </button>
    </div>
    <ProjectWebhookForm
      class="pt-4"
      :allowEdit="allowEdit"
      :create="false"
      :project="project"
      :webhook="projectWebhook"
    />
  </div>
</template>

<script lang="ts">
import { computed } from "@vue/runtime-core";
import ProjectWebhookForm from "../components/ProjectWebhookForm.vue";
import { idFromSlug } from "../utils";
import { useStore } from "vuex";
import {
  ProjectWebhookTestResult,
  PROJECT_HOOK_TYPE_ITEM_LIST,
} from "../types";

export default {
  name: "ProjectWebhookDetail",
  props: {
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
  },
  components: { ProjectWebhookForm },
  setup(props, ctx) {
    const store = useStore();

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const projectWebhook = computed(() => {
      return store.getters["projectWebhook/projectWebhookById"](
        idFromSlug(props.projectSlug),
        idFromSlug(props.projectWebhookSlug)
      );
    });

    const logo = computed(() => {
      for (const item of PROJECT_HOOK_TYPE_ITEM_LIST) {
        if (item.type == projectWebhook.value.type) {
          return item.logo;
        }
      }

      return "";
    });

    const testWebhook = () => {
      store
        .dispatch("projectWebhook/testProjectWebhookById", {
          projectId: idFromSlug(props.projectSlug),
          projectWebhookId: idFromSlug(props.projectWebhookSlug),
        })
        .then((testResult: ProjectWebhookTestResult) => {
          if (testResult.error) {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "CRITICAL",
              title: `Test webhook event failed.`,
              description: testResult.error,
              manualHide: true,
            });
          } else {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "SUCCESS",
              title: `Test webhook event OK.`,
            });
          }
        });
    };

    return {
      project,
      projectWebhook,
      logo,
      testWebhook,
    };
  },
};
</script>
