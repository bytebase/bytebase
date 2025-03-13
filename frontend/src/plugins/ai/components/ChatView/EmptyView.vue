<template>
  <div class="w-full h-full flex flex-col justify-center items-center">
    <div class="w-[20rem] mx-auto">
      <p>
        <NEmpty class="py-6" />
      </p>
      <p class="text-center textinfolabel">
        {{ $t("plugin.ai.conversation.no-message") }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NEmpty } from "naive-ui";
</script>
