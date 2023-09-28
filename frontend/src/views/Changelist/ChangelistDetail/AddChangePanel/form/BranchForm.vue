<template>
  <h1>add change from branch: {{ changes }}</h1>
</template>

<script setup lang="ts">
import { useAddChangeContext } from "../context";

const { changesFromBranch: changes } = useAddChangeContext();
</script>
