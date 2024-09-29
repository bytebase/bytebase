<template>
  <div class="flex flex-row items-center justify-between gap-x-4">
    <div class="flex-1 p-0.5 overflow-hidden truncate">
      <TitleEditor />
    </div>
    <div class="flex items-center justify-end gap-x-3">
      <ApplyToDatabaseButton />
    </div>
  </div>
</template>

<script setup lang="ts">
import ApplyToDatabaseButton from "./ApplyToDatabaseButton.vue";
import TitleEditor from "./TitleEditor.vue";
</script>
