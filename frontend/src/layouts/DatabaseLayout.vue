<template>
  <router-view :database-slug="databaseSlug" />
</template>

<script lang="ts" setup>
import { provideDatabaseDetailContext } from "@/components/Database/context";

const props = defineProps<{
  databaseSlug: string;
}>();

provideDatabaseDetailContext(props.databaseSlug);
</script>
