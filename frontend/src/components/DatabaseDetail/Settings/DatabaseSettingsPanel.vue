<template>
  <div class="flex flex-col">
    <Secrets :database="database" />
  </div>
</template>

<script setup lang="ts">
import { type Database } from "@/types";

defineProps<{
  database: Database;
}>();
</script>
