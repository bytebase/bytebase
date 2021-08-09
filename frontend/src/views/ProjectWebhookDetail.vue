<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="flex flex-row space-x-2 items-center">
      <img class="h-6 w-6" :src="`/src/assets/${logo}`" />
      <h3 class="text-xl leading-6 font-medium text-main">
        {{ projectHook.name }}
      </h3>
    </div>
    <ProjectWebhookForm
      class="pt-4"
      :allowEdit="allowEdit"
      :create="false"
      :project="project"
      :webhook="projectHook"
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
    projectHookSlug: {
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

    const projectHook = computed(() => {
      return store.getters["projectHook/projectHookById"](
        idFromSlug(props.projectSlug),
        idFromSlug(props.projectHookSlug)
      );
    });

    const logo = computed(() => {
      for (const item of PROJECT_HOOK_TYPE_ITEM_LIST) {
        if (item.type == projectHook.value.type) {
          return item.logo;
        }
      }

      return "";
    });

    return {
      project,
      projectHook,
      logo,
    };
  },
};
</script>
