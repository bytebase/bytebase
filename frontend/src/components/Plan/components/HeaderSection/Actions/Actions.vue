<template>
  <CreateButton v-if="isCreating" />
  <CreateIssueButton v-else-if="!plan.issue" />
</template>

<script setup lang="ts">
import { usePlanContext } from "@/components/Plan/logic";
import { CreateButton, CreateIssueButton } from "./create";

const { isCreating, plan } = usePlanContext();
</script>
