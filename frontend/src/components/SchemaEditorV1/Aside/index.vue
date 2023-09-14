<template>
  <BranchTree v-if="schemaEditorV1Store.resourceType === 'branch'" />
</template>

<script lang="ts" setup>
import { useSchemaEditorV1Store } from "@/store";
import BranchTree from "./BranchTree.vue";

const schemaEditorV1Store = useSchemaEditorV1Store();
</script>
