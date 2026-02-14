<template>
  <div v-if="available">
    <PermissionGuardWrapper
      v-slot="slotProps"
      :project="project"
      :permissions="[requiredPermission]"
    >
      <NButton
        type="primary"
        :text="text"
        :size="size"
        :disabled="slotProps.disabled || !hasRequestFeature"
        @click="onClick"
      >
        <template #icon>
          <ShieldUserIcon v-if="hasRequestFeature" class="w-4 h-4" />
          <FeatureBadge v-else :clickable="false" :feature="requiredFeature" />
        </template>
        {{ useJIT ? $t("sql-editor.request-jit") : $t("sql-editor.request-query") }}
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
    <AccessGrantRequestDrawer
      v-if="showJITDrawer"
      :query="statement"
      :targets="missingResources.map(r => r.databaseFullName)"
      @close="showJITDrawer = false"
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
import {
  type DatabaseResource,
  type Permission,
  PresetRoleType,
} from "@/types";
import { type PermissionDeniedDetail } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import AccessGrantRequestDrawer from "@/views/sql-editor/AsidePanel/AccessPane/AccessGrantRequestDrawer.vue";

const props = withDefaults(
  defineProps<{
    size?: "tiny" | "medium";
    text: boolean;
    preferJit: boolean;
    statement?: string;
    permissionDeniedDetail: PermissionDeniedDetail;
  }>(),
  {
    size: "medium",
  }
);

const showPanel = ref(false);
const showJITDrawer = ref(false);
const editorStore = useSQLEditorStore();
const projectStore = useProjectV1Store();

const project = computed(() =>
  projectStore.getProjectByName(editorStore.project)
);

const useJIT = computed(() => {
  return (
    props.preferJit &&
    project.value.allowJustInTimeAccess &&
    props.permissionDeniedDetail.requiredPermissions.every((p) =>
      p === "bb.sql.select"
    )
  );
});

const requiredPermission = computed(
  (): Permission =>
    useJIT.value ? "bb.accessGrants.create" : "bb.issues.create"
);

const requiredFeature = computed(() =>
  useJIT.value
    ? PlanFeature.FEATURE_JIT
    : PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW
);

const hasRequestFeature = computed(() => hasFeature(requiredFeature.value));

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

const available = computed(() => {
  return project.value.allowRequestRole;
});

const onClick = (e: MouseEvent) => {
  e.stopPropagation();
  e.preventDefault();

  if (useJIT.value) {
    showJITDrawer.value = true;
  } else {
    showPanel.value = true;
  }
};
</script>
