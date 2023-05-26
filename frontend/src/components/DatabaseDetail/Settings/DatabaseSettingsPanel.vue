<template>
  <div class="flex flex-col">
    <Secrets :database="database" />
  </div>
</template>

<script setup lang="ts">
import { type ComposedDatabase } from "@/types";
import Secrets from "./components/Secrets.vue";

defineProps<{
  database: ComposedDatabase;
}>();
</script>
