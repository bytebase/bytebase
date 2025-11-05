<template>
  <ProjectSettingPanel :project="project" :allow-edit="allowEdit" />
</template>

<script setup lang="ts">
import { computed, watchEffect } from "vue";
import ProjectSettingPanel from "@/components/ProjectSettingPanel.vue";
import { useBodyLayoutContext } from "@/layouts/common";
import { usePolicyV1Store, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const policyV1Store = usePolicyV1Store();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const preparePolicies = () => {
  policyV1Store.fetchPolicies({
    parent: project.value.name,
    resourceType: PolicyResourceType.PROJECT,
  });
};

watchEffect(preparePolicies);

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("py-0!");
</script>
