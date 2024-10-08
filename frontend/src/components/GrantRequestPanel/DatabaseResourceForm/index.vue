<template>
  <div class="w-full mb-2">
    <NRadioGroup
      v-model:value="state.allDatabases"
      class="w-full !flex flex-row justify-start items-center gap-4"
    >
      <NTooltip trigger="hover">
        <template #trigger>
          <NRadio
            :value="true"
            :label="$t('issue.grant-request.all-databases')"
          />
        </template>
        {{ $t("issue.grant-request.all-databases-tip") }}
      </NTooltip>
      <NRadio class="!leading-6" :value="false" :disabled="!project">
        <div class="flex items-center space-x-2">
          <FeatureBadge feature="bb.feature.access-control" />
          <span>{{ $t("issue.grant-request.manually-select") }}</span>
        </div>
      </NRadio>
    </NRadioGroup>
  </div>
  <div
    v-if="!state.allDatabases"
    class="w-full flex flex-row justify-start items-center"
  >
    <DatabaseResourceSelector
      v-if="project"
      :project-name="project.name"
      :database-resources="state.databaseResources"
      @update="handleSelectedDatabaseResourceChanged"
    />
  </div>
  <FeatureModal
    :open="state.showFeatureModal"
    :feature="'bb.feature.access-control'"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NRadioGroup, NRadio, NTooltip } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { useProjectByName, featureToRef } from "@/store";
import type { DatabaseResource } from "@/types";
import { stringifyDatabaseResources } from "@/utils/issue/cel";
import DatabaseResourceSelector from "./DatabaseResourceSelector.vue";

interface LocalState {
  allDatabases: boolean;
  showFeatureModal: boolean;
  databaseResources: DatabaseResource[];
}

const props = defineProps<{
  projectName: string;
  databaseResources?: DatabaseResource[];
}>();

const emit = defineEmits<{
  (event: "update:condition", value?: string): void;
  (
    event: "update:database-resources",
    databaseResources: DatabaseResource[]
  ): void;
}>();

const state = reactive<LocalState>({
  allDatabases: (props.databaseResources || []).length === 0,
  showFeatureModal: false,
  databaseResources: props.databaseResources || [],
});

const { project } = useProjectByName(computed(() => props.projectName));
const hasAccessControlFeature = featureToRef("bb.feature.access-control");

const handleSelectedDatabaseResourceChanged = (
  databaseResourceList: DatabaseResource[]
) => {
  state.databaseResources = databaseResourceList;
};

onMounted(() => {
  if (props.databaseResources && props.databaseResources.length > 0) {
    state.allDatabases = false;
  }
});

watch(
  () => [state.allDatabases, state.databaseResources],
  () => {
    if (state.allDatabases) {
      emit("update:condition", "");
    } else {
      if (!hasAccessControlFeature.value) {
        state.showFeatureModal = true;
        state.allDatabases = true;
        return;
      }
      if (state.databaseResources.length === 0) {
        emit("update:condition", undefined);
      } else {
        const condition = stringifyDatabaseResources(state.databaseResources);
        emit("update:condition", condition);
      }
      emit("update:database-resources", state.databaseResources);
    }
  },
  {
    immediate: true,
  }
);
</script>
