<template>
  <SchemaDiagram
    :database="database"
    :database-metadata="dbSchemaStore.getDatabaseMetadata(database.name)"
  />
</template>

<script setup lang="ts">
import SchemaDiagram from "@/components/SchemaDiagram";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";

const dbSchemaStore = useDBSchemaV1Store();
const { database } = useConnectionOfCurrentSQLEditorTab();
</script>

<style lang="postcss" scoped>
:deep(.bb-schema-diagram--navigator--main) {
  @apply border-0;
}
</style>
