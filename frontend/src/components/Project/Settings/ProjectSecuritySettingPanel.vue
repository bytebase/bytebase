<template>
  <div class="w-full flex flex-col justify-start items-start pt-6 space-y-4">
    <h3 class="text-lg font-medium leading-7 text-main">
      {{ $t("settings.sidebar.security-and-policy") }}
    </h3>
    <SQLReviewForResource
      v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
      :resource="project.name"
      :allow-edit="allowEdit"
    />
    <RestrictIssueCreationConfigure
      v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
      :resource="project.name"
      :allow-edit="allowEdit"
    />
    <AccessControlConfigure :resource="project.name" :allow-edit="allowEdit" />
  </div>
</template>

<script setup lang="ts">
import AccessControlConfigure from "@/components/EnvironmentForm/AccessControlConfigure.vue";
import RestrictIssueCreationConfigure from "@/components/GeneralSetting/RestrictIssueCreationConfigure.vue";
import { SQLReviewForResource } from "@/components/SQLReview";
import { useAppFeature } from "@/store";
import type { ComposedProject } from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";

defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
</script>
