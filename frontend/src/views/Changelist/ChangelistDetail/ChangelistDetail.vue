<template>
  <h1>detail</h1>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useChangelistStore } from "@/store";
import { unknownChangelist } from "@/types";

const route = useRoute();
const name = computed(() => {
  return `projects/${route.params["projectName"]}/changelists/${route.params["changelistName"]}`;
});
const changelist = computed(() => {
  return (
    useChangelistStore().getChangelistByName(name.value) ?? unknownChangelist()
  );
});

const documentTitle = computed(() => {
  return changelist.value.description;
});

useTitle(documentTitle);
</script>
