<template>
  <div>{{ project.name }}</div>
</template>

<script lang="ts">
import { computed } from "@vue/runtime-core";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";

export default {
  name: "ProjectWebhookCreate",
  props: {
    projectSlug: {
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

    return { project };
  },
};
</script>
