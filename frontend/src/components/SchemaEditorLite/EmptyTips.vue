<template>
  <div
    class="w-full h-full flex flex-row justify-center items-center text-gray-600 font-medium"
  >
    <SquarePen class="w-4 h-auto mr-2" />
    {{ $t("schema-designer.use-tips") }}
  </div>
</template>

<script lang="ts" setup>
import { SquarePen } from "lucide-vue-next";
</script>
