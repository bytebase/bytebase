<template>
  <div class="w-full flex flex-col justify-start items-start pt-6 space-y-4">
    <h3 class="text-lg font-medium leading-7 text-main">
      {{ $t("settings.sidebar.security-and-policy") }}
    </h3>
    <SQLReviewForResource :resource="project.name" :allow-edit="allowEdit" />
    <RestrictIssueCreationConfigure
      :resource="project.name"
      :allow-edit="allowEdit"
    />
    <AccessControlConfigure :resource="project.name" :allow-edit="allowEdit" />
  </div>
</template>

<script setup lang="ts">
import type { ComposedProject } from "@/types";

defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();
</script>
