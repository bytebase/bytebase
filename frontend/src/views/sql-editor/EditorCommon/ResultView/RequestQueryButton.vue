<template>
  <div v-if="available">
    <PermissionGuardWrapper
      v-slot="slotProps"
      :project="project"
      :permissions="['bb.issues.create']"
    >
      <NButton
        type="primary"
        :text="text"
        :size="size"
        :disabled="slotProps.disabled || !hasRequestRoleFeature"
        @click="onClick"
      >
        <template #icon>
          <ShieldUserIcon v-if="hasRequestRoleFeature" class="w-4 h-4" />
          <FeatureBadge v-else :clickable="false" :feature="PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW" />
        </template>
        {{ $t("sql-editor.request-query") }}
      </NButton>
    </PermissionGuardWrapper>

    <GrantRequestPanel
      v-if="showPanel"
      :project-name="project.name"
      :database-resources="missingResources"
      :placement="'right'"
      :role="PresetRoleType.SQL_EDITOR_USER"
      :required-permissions="permissionDeniedDetail.requiredPermissions"
      @close="showPanel = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ShieldUserIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { FeatureBadge } from "@/components/FeatureGuard";
import GrantRequestPanel from "@/components/GrantRequestPanel";
import { parseStringToResource } from "@/components/GrantRequestPanel/DatabaseResourceForm/common";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { hasFeature, useProjectV1Store, useSQLEditorStore } from "@/store";
import { type DatabaseResource, PresetRoleType } from "@/types";
import { type PermissionDeniedDetail } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const props = withDefaults(
  defineProps<{
    size?: "tiny" | "medium";
    text: boolean;
    permissionDeniedDetail: PermissionDeniedDetail;
  }>(),
  {
    size: "medium",
  }
);

const showPanel = ref(false);
const editorStore = useSQLEditorStore();
const projectStore = useProjectV1Store();
const hasRequestRoleFeature = computed(() =>
  hasFeature(PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW)
);

const missingResources = computed((): DatabaseResource[] => {
  const resources: DatabaseResource[] = [];
  for (const resourceString of props.permissionDeniedDetail.resources) {
    const resource = parseStringToResource(resourceString);
    if (resource) {
      resources.push(resource);
    }
  }
  return resources;
});

const project = computed(() =>
  projectStore.getProjectByName(editorStore.project)
);

const available = computed(() => {
  return project.value.allowRequestRole;
});

const onClick = (e: MouseEvent) => {
  e.stopPropagation();
  e.preventDefault();
  showPanel.value = true;
};
</script>
