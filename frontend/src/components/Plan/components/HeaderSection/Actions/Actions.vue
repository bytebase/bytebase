<template>
  <div class="h-full flex flex-row justify-end items-center">
    <CreateButton v-if="isCreating" />
    <CreateIssueButton v-else-if="!relatedIssueUID" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import { extractIssueUID } from "@/utils";
import { CreateButton, CreateIssueButton } from "./create";

const { isCreating, plan } = usePlanContext();
const relatedIssueUID = computed(() => extractIssueUID(plan.value.issue));
</script>
