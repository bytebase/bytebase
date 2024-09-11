<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <BytebaseLogo class="mx-auto mb-8" />

      <h2 class="text-2xl leading-9 font-medium text-main">
        {{ $t("setup.choose-workspace-mode") }}
      </h2>
    </div>

    <div class="mt-8"></div>
  </div>

  <AuthFooter />
</template>

<script lang="ts" setup>
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import AuthFooter from "./auth/AuthFooter.vue";
</script>
