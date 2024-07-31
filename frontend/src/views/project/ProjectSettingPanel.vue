<template>
  <ProjectSettingPanel :project="project" :allow-edit="allowEdit" />
</template>

<script setup lang="ts">
import { computed, watchEffect } from "vue";
import ProjectSettingPanel from "@/components/ProjectSettingPanel.vue";
import { usePolicyV1Store, useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PolicyResourceType } from "@/types/proto/v1/org_policy_service";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const policyV1Store = usePolicyV1Store();
const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const preparePolicies = () => {
  policyV1Store.fetchPolicies({
    parent: project.value.name,
    resourceType: PolicyResourceType.PROJECT,
  });
};

watchEffect(preparePolicies);
</script>
