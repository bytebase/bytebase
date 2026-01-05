<template>
  <div class="flex flex-col gap-y-4">
    <FeatureAttention :feature="PlanFeature.FEATURE_DATABASE_GROUPS" />

    <div class="flex flex-row items-center justify-end gap-x-2">
      <SearchBox
        v-model:value="state.searchText"
        style="max-width: 100%"
        :placeholder="$t('common.filter-by-name')"
      />
      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="['bb.databaseGroups.create']"
      >
        <NButton
          type="primary"
          :disabled="slotProps.disabled"
          @click="
            () => {
              if (!hasDBGroupFeature) {
                state.showFeatureModal = true;
                return;
              }
              router.push({
                name: PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
              });
            }
          "
        >
          <template #icon>
            <PlusIcon class="w-4" />
            <FeatureBadge
              :feature="PlanFeature.FEATURE_DATABASE_GROUPS"
              class="text-white"
            />
          </template>
          {{ $t("database-group.create") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>
    <ProjectDatabaseGroupPanel :project="project" :filter="state.searchText" />
  </div>

  <FeatureModal
    :open="state.showFeatureModal"
    :feature="PlanFeature.FEATURE_DATABASE_GROUPS"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import ProjectDatabaseGroupPanel from "@/components/DatabaseGroup/ProjectDatabaseGroupPanel.vue";
import {
  FeatureAttention,
  FeatureBadge,
  FeatureModal,
} from "@/components/FeatureGuard";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { SearchBox } from "@/components/v2";
import { PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE } from "@/router/dashboard/projectV1";
import { hasFeature, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  showFeatureModal: boolean;
  searchText: string;
}

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const router = useRouter();
const state = reactive<LocalState>({
  showFeatureModal: false,
  searchText: "",
});

const hasDBGroupFeature = computed(() =>
  hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)
);
</script>
