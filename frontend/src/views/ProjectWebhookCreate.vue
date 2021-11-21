<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="text-xl leading-6 font-medium text-main">Create webhook</div>
    <ProjectWebhookForm
      class="pt-4"
      :create="true"
      :project="project"
      :webhook="DEFAULT_NEW_WEBHOOK"
    />
  </div>
</template>

<script lang="ts">
import { computed } from "@vue/runtime-core";
import { useStore } from "vuex";
import ProjectWebhookForm from "../components/ProjectWebhookForm.vue";
import { idFromSlug } from "../utils";
import { ProjectWebhookCreate } from "../types";

const DEFAULT_NEW_WEBHOOK: ProjectWebhookCreate = {
  type: "bb.plugin.webhook.slack",
  name: "",
  url: "",
  activityList: ["bb.issue.status.update"],
};

export default {
  name: "ProjectWebhookCreate",
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
  },
  components: { ProjectWebhookForm },
  setup(props, ctx) {
    const store = useStore();

    const project = computed(() => {
      return store.getters["project/projectByID"](
        idFromSlug(props.projectSlug)
      );
    });

    return {
      DEFAULT_NEW_WEBHOOK,
      project,
    };
  },
};
</script>
