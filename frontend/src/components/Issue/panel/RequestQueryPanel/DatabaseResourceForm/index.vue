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
      <NRadio
        class="!leading-6"
        :value="false"
        :disabled="!project"
        :label="$t('issue.grant-request.manually-select')"
      />
    </NRadioGroup>
  </div>
  <div
    v-if="!state.allDatabases"
    class="w-full flex flex-row justify-start items-center"
  >
    <DatabaseResourceSelector
      v-if="project"
      :project-id="project.uid"
      :database-resources="state.databaseResources"
      @update="handleSelectedDatabaseResourceChanged"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch, watchEffect } from "vue";
import { useProjectV1Store } from "@/store";
import { DatabaseResource } from "@/types";
import { stringifyDatabaseResources } from "@/utils/issue/cel";
import DatabaseResourceSelector from "./DatabaseResourceSelector.vue";

interface LocalState {
  allDatabases: boolean;
  databaseResources: DatabaseResource[];
}

const props = defineProps<{
  projectId?: string;
  databaseResources?: DatabaseResource[];
}>();

const emit = defineEmits<{
  (event: "update:condition", value?: string): void;
  (
    event: "update:database-resources",
    databaseResources: DatabaseResource[]
  ): void;
}>();

const projectStore = useProjectV1Store();
const state = reactive<LocalState>({
  allDatabases: (props.databaseResources || []).length === 0,
  databaseResources: props.databaseResources || [],
});

const project = computed(() => {
  return props.projectId
    ? projectStore.getProjectByUID(props.projectId)
    : undefined;
});

// Prepare project entity.
watchEffect(async () => {
  if (!props.projectId) {
    return;
  }
  await projectStore.getOrFetchProjectByUID(props.projectId);
});

const handleSelectedDatabaseResourceChanged = (
  databaseResourceList: DatabaseResource[]
) => {
  state.databaseResources = databaseResourceList;
};

watch(
  () => [state.allDatabases, state.databaseResources],
  () => {
    if (state.allDatabases) {
      emit("update:condition", "");
    } else {
      if (state.databaseResources.length === 0) {
        emit("update:condition", undefined);
      } else {
        const condition = stringifyDatabaseResources(state.databaseResources);
        emit("update:condition", condition);
      }
      emit("update:database-resources", state.databaseResources);
    }
  }
);
</script>
