<template>
  <CreateButton v-if="isCreating" />
  <CreateIssueButton
    v-else-if="
      databaseChangeMode === DatabaseChangeMode.PIPELINE && !relatedIssueUID
    "
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import { useAppFeature } from "@/store";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { extractIssueUID } from "@/utils";
import { CreateButton, CreateIssueButton } from "./create";

const { isCreating, plan } = usePlanContext();
const relatedIssueUID = computed(() => extractIssueUID(plan.value.issue));
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
</script>
