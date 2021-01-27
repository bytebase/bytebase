<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="relative h-screen overflow-hidden flex flex-col">
    <nav class="bg-white border-b border-block-border">
      <div class="max-w-full mx-auto px-6">
        <DashboardHeader />
      </div>
    </nav>

    <router-view name="body" />
  </div>
</template>

<script lang="ts">
import DashboardHeader from "../views/DashboardHeader.vue";

export default {
  name: "DashboardLayout",
  components: { DashboardHeader },
  setup(props, ctx) {},
};
</script>
