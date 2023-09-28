<template>
  <div class="flex flex-col gap-y-2 px-4">
    <NavBar />
    <h1>detail</h1>

    <AddChangePanel />
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed } from "vue";
import AddChangePanel from "./AddChangePanel";
import NavBar from "./NavBar";
import { provideChangelistDetailContext } from "./context";

const { changelist } = provideChangelistDetailContext();

const documentTitle = computed(() => {
  return changelist.value.description;
});

useTitle(documentTitle);
</script>
