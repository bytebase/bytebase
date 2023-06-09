<template>
  <UserTable
    :project="project"
    :iam-policy="iamPolicy"
    :ready="ready"
    :editable="editable"
    :member-list="memberList"
  />
</template>

<script lang="ts" setup>
import type { IamPolicy, Project } from "@/types/proto/v1/project_service";
import UserTable from "./UserTable.vue";
import { ComposedProjectMember } from "./types";

defineProps<{
  project: Project;
  iamPolicy: IamPolicy;
  ready?: boolean;
  editable: boolean;
  memberList: ComposedProjectMember[];
}>();
</script>
