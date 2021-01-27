<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="relative h-screen flex flex-col">
    <nav class="bg-white border-b border-block-border">
      <div class="max-w-full mx-auto px-6">
        <DashboardHeader />
      </div>
    </nav>

    <main class="flex flex-grow">
      <div class="max-w-full">
        <router-view name="body" />
      </div>
    </main>
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
