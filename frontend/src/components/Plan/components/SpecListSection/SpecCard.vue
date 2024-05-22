<template>
  <div
    class="px-2 py-1 pr-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-stretch overflow-hidden gap-x-1"
    :class="specClass"
    @click="onClickSpec(spec)"
  >
    <div
      v-if="isDatabaseChangeSpec(spec)"
      class="w-full flex items-center gap-2"
    >
      <InstanceV1Name
        :instance="databaseForSpec(plan, spec).instanceEntity"
        :link="false"
        class="text-gray-500 text-sm"
      />
      <span class="truncate text-sm">{{
        databaseForSpec(plan, spec).databaseName
      }}</span>
    </div>
    <div
      v-else-if="isGroupingChangeSpec(spec) && relatedDatabaseGroup"
      class="w-full flex items-center gap-2"
    >
      <NTooltip>
        <template #trigger><DatabaseGroupIcon class="w-4 h-auto" /></template>
        {{ $t("resource.database-group") }}
      </NTooltip>
      <span class="truncate text-sm">{{
        relatedDatabaseGroup.databaseGroupName
      }}</span>
    </div>
    <div
      v-else-if="isDeploymentConfigChangeSpec(spec)"
      class="w-full flex items-center"
    >
      <TenantIcon class="w-4 h-auto" />
      <span class="text-gray-500 text-sm truncate ml-1 mr-2">
        {{ project.title }}
      </span>
      <span class="text-sm">{{ $t("common.deployment-config") }}</span>
    </div>
    <!-- Fallback -->
    <div v-else class="w-full flex items-center gap-2 text-sm">
      Unknown type
    </div>
  </div>
</template>

<script setup lang="ts">
import { isEqual } from "lodash-es";
import { NTooltip } from "naive-ui";
import { computed, onMounted } from "vue";
import DatabaseGroupIcon from "@/components/DatabaseGroupIcon.vue";
import TenantIcon from "@/components/TenantIcon.vue";
import { useDBGroupStore } from "@/store";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import {
  databaseForSpec,
  isDatabaseChangeSpec,
  usePlanContext,
  isGroupingChangeSpec,
  isDeploymentConfigChangeSpec,
} from "../../logic";

const props = defineProps<{
  spec: Plan_Spec;
}>();

const { plan, selectedSpec, events } = usePlanContext();
const dbGroupStore = useDBGroupStore();

const project = computed(() => plan.value.projectEntity);

const specClass = computed(() => {
  const classes: string[] = [];
  if (isEqual(props.spec, selectedSpec.value)) {
    classes.push("border-accent bg-accent bg-opacity-5 shadow");
  }
  return classes;
});

const relatedDatabaseGroup = computed(() => {
  if (!isGroupingChangeSpec(props.spec)) {
    return undefined;
  }
  return dbGroupStore.getDBGroupByName(props.spec.changeDatabaseConfig!.target);
});

onMounted(async () => {
  if (isGroupingChangeSpec(props.spec)) {
    await dbGroupStore.getOrFetchDBGroupByName(
      props.spec.changeDatabaseConfig!.target
    );
  }
});

const onClickSpec = (spec: Plan_Spec) => {
  events.emit("select-spec", { spec });
};
</script>
