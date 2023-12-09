<template>
  <BranchTree v-if="resourceType === 'branch'" />
  <DatabaseTree v-else-if="resourceType === 'database'" />
</template>

<script lang="ts" setup>
import { useSchemaEditorContext } from "../context";
import BranchTree from "./BranchTree.vue";
import DatabaseTree from "./DatabaseTree.vue";

const { resourceType } = useSchemaEditorContext();
</script>
