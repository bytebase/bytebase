<template>
  <router-view />
</template>

<script lang="ts">
import { onErrorCaptured } from "vue";
import { useStore } from "vuex";

export default {
  name: "App",
  components: {},
  setup(props, ctx) {
    const store = useStore();

    onErrorCaptured((e: any, _, info) => {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: `Internal error occured in '${info}'.`,
        description: e.stack,
      });
      return true;
    });
  },
};
</script>
