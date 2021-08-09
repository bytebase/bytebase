<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="flex flex-row space-x-2 items-center">
      <img class="h-6 w-6" :src="`/src/assets/${logo}`" />
      <h3 class="text-xl leading-6 font-medium text-main">
        {{ projectWebhook.name }}
      </h3>
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
import { PROJECT_HOOK_TYPE_ITEM_LIST } from "../types";

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

    return {
      project,
      projectWebhook,
      logo,
    };
  },
};
</script>
