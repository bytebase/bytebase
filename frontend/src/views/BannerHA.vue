<template>
  <div class="bg-warning flex items-center justify-center p-3">
    <TriangleAlertIcon
      class="h-6 w-6 text-white shrink-0"
    />
    <p class="ml-3 font-medium text-white truncate">
      <span>
        {{ $t("banner.ha-not-licensed") }}
      </span>
    </p>
  </div>
</template>

<script lang="ts" setup>
import { TriangleAlertIcon } from "lucide-vue-next";
</script>
