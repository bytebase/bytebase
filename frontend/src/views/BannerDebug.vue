<template>
  <div class="bg-warning">
    <div class="text-center py-3 px-3 font-medium text-white truncate">
      {{ $t("banner.debug") }}
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  name: "BannerDebug",
});
</script>
