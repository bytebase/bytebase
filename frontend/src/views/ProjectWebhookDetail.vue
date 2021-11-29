<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="flex justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
        <template v-if="projectWebhook.type == 'bb.plugin.webhook.slack'">
          <img class="h-6 w-6" src="../assets/slack-logo.png" alt="" />
        </template>
        <template
          v-else-if="projectWebhook.type == 'bb.plugin.webhook.discord'"
        >
          <img class="h-6 w-6" src="../assets/discord-logo.svg" />
        </template>
        <template v-else-if="projectWebhook.type == 'bb.plugin.webhook.teams'">
          <img class="h-6 w-6" src="../assets/teams-logo.svg" />
        </template>
        <template
          v-else-if="projectWebhook.type == 'bb.plugin.webhook.dingtalk'"
        >
          <img class="h-6 w-6" src="../assets/dingtalk-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type == 'bb.plugin.webhook.feishu'">
          <img class="h-6 w-6" src="../assets/feishu-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type == 'bb.plugin.webhook.wecom'">
          <img class="h-6 w-6" src="../assets/wecom-logo.png" />
        </template>
        <h3 class="text-xl leading-6 font-medium text-main">
          {{ projectWebhook.name }}
        </h3>
      </div>
      <button
        type="button"
        class="btn-normal whitespace-nowrap items-center"
        @click.prevent="testWebhook"
      >
        Test Webhook
      </button>
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

<script lang="ts">
import { computed } from "@vue/runtime-core";
import ProjectWebhookForm from "../components/ProjectWebhookForm.vue";
import { idFromSlug } from "../utils";
import { useStore } from "vuex";
import { ProjectWebhookTestResult } from "../types";

export default {
  name: "ProjectWebhookDetail",
  components: { ProjectWebhookForm },
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
  setup(props) {
    const store = useStore();

    const project = computed(() => {
      return store.getters["project/projectByID"](
        idFromSlug(props.projectSlug)
      );
    });

    const projectWebhook = computed(() => {
      return store.getters["projectWebhook/projectWebhookByID"](
        idFromSlug(props.projectSlug),
        idFromSlug(props.projectWebhookSlug)
      );
    });

    const testWebhook = () => {
      store
        .dispatch("projectWebhook/testProjectWebhookByID", {
          projectID: idFromSlug(props.projectSlug),
          projectWebhookID: idFromSlug(props.projectWebhookSlug),
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
      testWebhook,
    };
  },
};
</script>
