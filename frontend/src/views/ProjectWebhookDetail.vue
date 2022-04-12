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
        {{ $t("project.webhook.test-webhook") }}
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
import { computed, defineComponent } from "vue";
import ProjectWebhookForm from "../components/ProjectWebhookForm.vue";
import { idFromSlug } from "../utils";
import { ProjectWebhookTestResult } from "../types";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useProjectWebhookStore,
  useProjectStore,
} from "@/store";

export default defineComponent({
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
    const { t } = useI18n();
    const projectWebhookStore = useProjectWebhookStore();
    const projectStore = useProjectStore();

    const project = computed(() => {
      return projectStore.getProjectById(idFromSlug(props.projectSlug));
    });

    const projectWebhook = computed(() => {
      return projectWebhookStore.projectWebhookById(
        idFromSlug(props.projectSlug),
        idFromSlug(props.projectWebhookSlug)
      );
    });

    const testWebhook = () => {
      projectWebhookStore
        .testProjectWebhookById({
          projectId: idFromSlug(props.projectSlug),
          projectWebhookId: idFromSlug(props.projectWebhookSlug),
        })
        .then((testResult: ProjectWebhookTestResult) => {
          if (testResult.error) {
            pushNotification({
              module: "bytebase",
              style: "CRITICAL",
              title: t("project.webhook.fail-tested-title"),
              description: testResult.error,
              manualHide: true,
            });
          } else {
            pushNotification({
              module: "bytebase",
              style: "SUCCESS",
              title: t("project.webhook.success-tested-prompt"),
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
});
</script>
