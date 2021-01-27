<template>
  <ProvideUser>
    <router-view />
  </ProvideUser>
</template>

<script lang="ts">
import ProvideUser from "./components/ProvideUser.vue";

export default {
  name: "App",
  components: { ProvideUser },
  setup(props, ctx) {},
};
</script>
