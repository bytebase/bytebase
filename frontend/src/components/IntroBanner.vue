<template>
  <div class="hidden md:flex justify-center items-center border m-4">
    <img class="m-4 h-72" src="../assets/illustration/guide.webp" alt="" />
    <div class="m-4">
      <h2 class="text-xl text-main">
        Bytebase is for DBAs and Developers to collaborate on database schemas
        changes. One can construct a single pipeline to propagate the schema
        change across multiple environments. One can also store the schemas in
        VCS and trigger a new pipeline upon commit push. Follow the
        <span class="text-accent">Quickstart</span> on the bottom left to get
        familiar with the product. If you encounter problems, visit
        <a class="normal-link" href="https://docs.bytebase.com" target="__blank"
          >doc</a
        >
        or report
        <a
          class="normal-link"
          href="https://github.com/bytebase/bytebase/issues"
          target="__blank"
          >issue</a
        >
        on GitHub.
      </h2>
      <div class="mt-4">
        <button class="btn-normal" @click.prevent="dismiss">Dismiss</button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { useStore } from "vuex";
export default {
  name: "IntroBanner",
  components: {},
  setup() {
    const store = useStore();
    const dismiss = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "general.overview",
        newState: true,
      });
    };
    return {
      dismiss,
    };
  },
};
</script>
