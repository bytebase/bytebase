<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="text-xl leading-6 font-medium text-main">
      {{ $t("project.webhook.creation.title") }}
    </div>
    <ProjectWebhookForm
      class="pt-4"
      :create="true"
      :project="project"
      :webhook="DEFAULT_NEW_WEBHOOK"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent } from "vue";
import ProjectWebhookForm from "../components/ProjectWebhookForm.vue";
import { idFromSlug } from "../utils";
import { ProjectWebhookCreate } from "../types";
import { useProjectStore } from "@/store";

const DEFAULT_NEW_WEBHOOK: ProjectWebhookCreate = {
  type: "bb.plugin.webhook.slack",
  name: "",
  url: "",
  activityList: ["bb.issue.status.update"],
};

export default defineComponent({
  name: "ProjectWebhookCreate",
  components: { ProjectWebhookForm },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props) {
    const projectStore = useProjectStore();

    const project = computed(() => {
      return projectStore.getProjectById(idFromSlug(props.projectSlug));
    });

    return {
      DEFAULT_NEW_WEBHOOK,
      project,
    };
  },
});
</script>
