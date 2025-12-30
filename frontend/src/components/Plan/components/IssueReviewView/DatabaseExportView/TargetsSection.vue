<template>
  <div class="flex flex-col gap-y-2">
    <template v-if="targets.length > 0">
      <h3 class="text-base font-medium">
        {{ $t("plan.targets.title") }}
      </h3>
      <div class="flex flex-wrap gap-2">
        <div
          v-for="target in targets"
          :key="target"
          class="inline-flex items-center gap-2 px-2 py-1.5 border rounded-sm min-w-0"
        >
          <template v-if="isValidDatabaseName(target)">
            <DatabaseDisplay
              :database="target"
              :show-environment="true"
              size="medium"
              class="flex-1 min-w-0"
            />
          </template>
          <template v-else-if="isValidDatabaseGroupName(target)">
            <DatabaseGroupTargetDisplay :target="target" class="px-1 py-1" />
          </template>
          <template v-else>
            <span class="text-sm">{{ target }}</span>
          </template>
        </div>
      </div>
    </template>
    <div v-else class="text-center text-control-light py-8">
      {{ $t("common.no-data") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import DatabaseGroupTargetDisplay from "@/components/Plan/components/SpecDetailView/DatabaseGroupTargetDisplay.vue";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { usePlanContext } from "../../../logic";

const { plan } = usePlanContext();
const dbStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();

const targets = computed(() => {
  const exportDataSpec = plan.value.specs.find(
    (spec) => spec.config?.case === "exportDataConfig"
  );
  if (exportDataSpec?.config.case === "exportDataConfig") {
    return exportDataSpec.config.value.targets || [];
  }
  return [];
});

// Fetch target data
watchEffect(() => {
  for (const target of targets.value) {
    if (isValidDatabaseName(target)) {
      dbStore.getOrFetchDatabaseByName(target);
    } else if (isValidDatabaseGroupName(target)) {
      dbGroupStore.getOrFetchDBGroupByName(target, {
        view: DatabaseGroupView.FULL,
      });
    }
  }
});
</script>
