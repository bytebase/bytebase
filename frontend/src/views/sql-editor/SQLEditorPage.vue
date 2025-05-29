<template>
  <Suspense>
    <template #default>
      <SQLEditorHomePage />
    </template>
    <template #fallback>
      <div class="flex items-center justify-center h-screen">
        <BBSpin />
      </div>
    </template>
  </Suspense>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from "vue";
import { BBSpin } from "@/bbkit";

// Lazy load the SQL Editor to reduce initial bundle size.
const SQLEditorHomePage = defineAsyncComponent({
  loader: () => import("./SQLEditorHomePage.vue"),
  delay: 200,
  timeout: 30000,
  errorComponent: {
    template: `
      <div class="flex items-center justify-center h-screen">
        <div class="text-red-500">Failed to load SQL Editor</div>
      </div>
    `,
  },
  onError(error) {
    console.error("Failed to load SQL Editor:", error);
  },
});
</script>
