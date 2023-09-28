<template>
  <div class="flex flex-col gap-y-4"></div>
  <h1>add change from raw SQL: {{ changes }}</h1>
</template>

<script setup lang="ts">
import { useAddChangeContext } from "../context";

const { changesFromRawSQL: changes } = useAddChangeContext();
</script>
