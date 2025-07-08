<template>
  <div class="w-full mb-2">
    <NRadioGroup
      v-if="allowSelectAll"
      v-model:value="state.allDatabases"
      :disabled="disabled"
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
        <div class="flex items-center space-x-1">
          <FeatureBadge :feature="requiredFeature" />
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
      v-model:database-resources="state.databaseResources"
      :disabled="disabled"
      :project-name="project.name"
      :include-cloumn="includeCloumn"
    />
  </div>
  <FeatureModal
    :open="state.showFeatureModal"
    :feature="requiredFeature"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NRadioGroup, NRadio, NTooltip } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { useProjectByName, hasFeature } from "@/store";
import type { DatabaseResource } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import DatabaseResourceSelector from "./DatabaseResourceSelector.vue";

interface LocalState {
  allDatabases: boolean;
  showFeatureModal: boolean;
  databaseResources: DatabaseResource[];
}

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    projectName: string;
    requiredFeature: PlanFeature;
    includeCloumn: boolean;
    allowSelectAll?: boolean;
    databaseResources?: DatabaseResource[];
  }>(),
  {
    disabled: false,
    allowSelectAll: true,
    databaseResources: undefined,
  }
);

const emit = defineEmits<{
  (
    event: "update:database-resources",
    databaseResources?: DatabaseResource[]
  ): void;
}>();

const state = reactive<LocalState>({
  allDatabases:
    props.allowSelectAll && (props.databaseResources || []).length === 0,
  showFeatureModal: false,
  databaseResources: props.databaseResources || [],
});

const { project } = useProjectByName(computed(() => props.projectName));
const hasRequiredFeature = computed(() => hasFeature(props.requiredFeature));

onMounted(() => {
  if (props.databaseResources && props.databaseResources.length > 0) {
    state.allDatabases = false;
  }
});

watch(
  () => [state.allDatabases, state.databaseResources],
  () => {
    if (state.allDatabases) {
      emit("update:database-resources", undefined);
    } else {
      if (!hasRequiredFeature.value) {
        state.showFeatureModal = true;
        state.allDatabases = true;
        return;
      }
      emit("update:database-resources", state.databaseResources);
    }
  },
  {
    immediate: true,
  }
);
</script>
