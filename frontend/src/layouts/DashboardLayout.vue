<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <nav class="bg-white border-b border-block-border">
    <div class="max-w-full mx-auto px-6">
      <DashboardHeader />
    </div>
  </nav>

  <main>
    <div class="max-w-full">
      <router-view />
    </div>
  </main>
</template>

<script lang="ts">
import { reactive } from "vue";
import DashboardHeader from "../views/DashboardHeader.vue";

export default {
  name: "DashboardLayout",
  components: { DashboardHeader },
  setup(props, ctx) {},
};
</script>
