<template>
  <div class="bg-white flex flex-col gap-y-2 w-max p-2">
    <div class="font-semibold whitespace-nowrap">
      {{ $t("sql-editor.customize-tree-structure") }}
    </div>
    <FactorTags class="pt-1 pb-2" />
  </div>
</template>

<script setup lang="ts">
import FactorTags from "./FactorTags.vue";
</script>
