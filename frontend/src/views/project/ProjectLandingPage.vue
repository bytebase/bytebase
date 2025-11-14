<template>
  <div />
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_ISSUES,
} from "@/router/dashboard/projectV1";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
const projectStore = useProjectV1Store();
const router = useRouter();

watchEffect(async () => {
  const project = await projectStore.getOrFetchProjectByName(projectName.value);
  if (hasProjectPermissionV2(project, "bb.issues.list")) {
    router.replace({
      name: PROJECT_V1_ROUTE_ISSUES,
    });
  } else {
    router.replace({
      name: PROJECT_V1_ROUTE_DATABASES,
    });
  }
});
</script>
