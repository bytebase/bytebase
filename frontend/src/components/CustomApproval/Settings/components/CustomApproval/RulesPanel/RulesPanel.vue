<template>
  <div class="w-full flex flex-col gap-y-4">
    <RulesSection
      v-for="source in supportedSourceList"
      :key="source"
      :source="source"
    />
  </div>
</template>

<script lang="ts" setup>
import { useSupportedSourceList } from "@/types";
import RulesSection from "./RulesSection.vue";

const supportedSourceList = useSupportedSourceList();
</script>
