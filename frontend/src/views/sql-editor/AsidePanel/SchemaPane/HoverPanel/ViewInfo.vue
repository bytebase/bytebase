<template>
  <CommonText :content="viewMetadata.comment" />
</template>

<script setup lang="tsx">
import { computed } from "vue";
import { useDBSchemaV1Store } from "@/store";
import CommonText from "./CommonText.vue";

const props = defineProps<{
  database: string;
  schema?: string;
  view: string;
}>();

const dbSchema = useDBSchemaV1Store();

const viewMetadata = computed(() =>
  dbSchema.getViewMetadata({
    database: props.database,
    schema: props.schema,
    view: props.view,
  })
);
</script>
