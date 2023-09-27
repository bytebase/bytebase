<template>
  <div class="flex flex-row items-center justify-between py-0.5">
    <div class="flex items-center justify-start">
      <ProjectV1Name :project="project" />
      <span class="ml-3 mr-0.5">/</span>
      <TitleEditor />
    </div>
    <div class="flex items-center justify-end gap-x-2">
      <NButton icon>
        <template #icon>
          <heroicons:arrows-up-down />
        </template>
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { ProjectV1Name } from "@/components/v2";
import { useChangelistDetailContext } from "../context";
import TitleEditor from "./TitleEditor.vue";

const { project } = useChangelistDetailContext();
</script>
