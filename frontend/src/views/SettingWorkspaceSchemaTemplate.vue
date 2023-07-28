<template>
  <template v-if="state.selectedTab === 'FIELD_TEMPLATE'">
    <FieldTemplates />
  </template>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";

interface LocalState {
  selectedTab: "FIELD_TEMPLATE";
}

const state = reactive<LocalState>({
  selectedTab: "FIELD_TEMPLATE",
});
</script>
