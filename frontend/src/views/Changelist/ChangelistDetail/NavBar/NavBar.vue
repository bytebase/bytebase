<template>
  <div class="flex flex-row items-center justify-between py-0.5">
    <div class="flex items-center justify-start">
      <ProjectV1Name :project="project" />
      <span class="ml-3 mr-0.5">/</span>
      <TitleEditor />
    </div>
    <div class="flex items-center justify-end gap-x-3">
      <NButton icon style="--n-padding: 0 10px">
        <template #icon>
          <heroicons:arrows-up-down />
        </template>
      </NButton>
      <NButton icon style="--n-padding: 0 10px">
        <template #icon>
          <heroicons:arrow-down-tray />
        </template>
      </NButton>
      <NButton icon style="--n-padding: 0 10px">
        <template #icon>
          <heroicons:plus />
        </template>
      </NButton>
      <NButton type="primary">
        {{ $t("changelist.apply-to-database") }}
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
