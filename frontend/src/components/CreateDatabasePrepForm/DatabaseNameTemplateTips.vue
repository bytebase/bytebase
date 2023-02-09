<template>
  <div>
    <p>{{ name }}</p>
    <p>{{ project.dbNameTemplate }}</p>
    <p>{{ labelList }}</p>
  </div>
</template>

<script lang="ts" setup>
import type { DatabaseLabel, Project } from "@/types";

defineProps<{
  name: string;
  project: Project;
  labelList: DatabaseLabel[];
}>();
</script>
