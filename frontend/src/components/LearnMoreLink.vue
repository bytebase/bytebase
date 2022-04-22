<template>
  <a class="normal-link inline-flex items-center" :href="url" target="__BLANK">
    {{ $t("common.learn-more") }}
    <heroicons-outline:external-link class="w-4 h-4 ml-1" />
  </a>
</template>

<script lang="ts" setup>
defineProps({
  url: {
    type: String,
    required: true,
  },
});
</script>
