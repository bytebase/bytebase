<template>
  <div class="w-full flex-1 flex">
    <SpecDetailView v-if="specId" :key="specId" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import { SpecDetailView } from "@/components/Plan/components";

const route = useRoute();

const specId = computed(() => route.params.specId as string | undefined);
</script>
