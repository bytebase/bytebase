<template>
  <PrincipalTable :project="project" :iam-policy="iamPolicy" :ready="ready" />
</template>

<script lang="ts" setup>
import { Project } from "@/types";
import { IamPolicy } from "@/types/proto/v1/project_service";
import PrincipalTable from "./PrincipalTable.vue";

defineProps<{
  project: Project;
  iamPolicy: IamPolicy;
  ready?: boolean;
}>();
</script>
