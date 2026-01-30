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
      :database-resources="databaseResources"
      :placement="'right'"
      :role="PresetRoleType.SQL_EDITOR_USER"
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
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { hasFeature, useProjectV1Store, useSQLEditorStore } from "@/store";
import { type DatabaseResource, PresetRoleType } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

withDefaults(
  defineProps<{
    databaseResources: DatabaseResource[];
    size?: "tiny" | "medium";
    text: boolean;
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
