<template>
  <BranchTree v-if="schemaEditorV1Store.resourceType === 'branch'" />
  <DatabaseTree v-else-if="schemaEditorV1Store.resourceType === 'database'" />
</template>

<script lang="ts" setup>
import { useSchemaEditorV1Store } from "@/store";
import BranchTree from "./BranchTree.vue";
import DatabaseTree from "./DatabaseTree.vue";

const schemaEditorV1Store = useSchemaEditorV1Store();
</script>
