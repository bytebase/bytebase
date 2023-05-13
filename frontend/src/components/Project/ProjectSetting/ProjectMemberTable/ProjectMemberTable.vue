<template>
  <PrincipalTable
    :project="project"
    :iam-policy="iamPolicy"
    :ready="ready"
    :editable="editable"
    :composed-principal-list="composedPrincipalList"
  />
</template>

<script lang="ts" setup>
import type { IamPolicy, Project } from "@/types/proto/v1/project_service";
import { ComposedPrincipal } from "@/types";
import PrincipalTable from "./PrincipalTable.vue";

defineProps<{
  project: Project;
  iamPolicy: IamPolicy;
  ready?: boolean;
  editable: boolean;
  composedPrincipalList: ComposedPrincipal[];
}>();
</script>
