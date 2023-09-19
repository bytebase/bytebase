<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="text-xl leading-6 font-medium text-main">
      {{ $t("project.webhook.creation.title") }}
    </div>
    <ProjectWebhookForm
      class="pt-4"
      :create="true"
      :project="project"
      :webhook="defaultNewWebhook"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent } from "vue";
import { useProjectV1Store } from "@/store";
import { emptyProjectWebhook } from "@/types";
import ProjectWebhookForm from "../components/ProjectWebhookForm.vue";
import { idFromSlug } from "../utils";

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
    const projectV1Store = useProjectV1Store();

    const project = computed(() => {
      return projectV1Store.getProjectByUID(
        String(idFromSlug(props.projectSlug))
      );
    });

    const defaultNewWebhook = emptyProjectWebhook();

    return {
      defaultNewWebhook,
      project,
    };
  },
});
</script>
