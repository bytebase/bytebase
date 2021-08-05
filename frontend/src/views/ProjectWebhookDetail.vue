<template>
  <div>{{ project.name }} - {{ projectHook.name }}</div>
</template>

<script lang="ts">
import { computed } from "@vue/runtime-core";
import { idFromSlug } from "../utils";
import { useStore } from "vuex";

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
  },
  components: {},
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

    return {
      project,
      projectHook,
    };
  },
};
</script>
