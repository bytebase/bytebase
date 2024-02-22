<template>
  <BBModal :title="title" class="w-full h-screen">
    <SchemaDiffViewer :title="''" :original="original" :modified="modified" />
  </BBModal>
</template>

<script lang="ts" setup>
import SchemaDiffViewer from "./SchemaDiffViewer.vue";

defineProps<{
  title: string;
  original: string;
  modified: string;
}>();
</script>
