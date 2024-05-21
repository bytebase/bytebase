<template>
  <div class="w-full flex flex-col divide-y">
    <template v-if="selectedSpec">
      <div class="w-full flex flex-row items-center py-2 px-2 sm:px-4">
        <span class="textlabel mr-4">{{ $t("common.database") }}</span>
        <DatabaseInfo :database="database" />
      </div>
      <div class="w-full py-2 px-2 sm:px-4">
        <ReviewOptionSection />
      </div>
    </template>
    <template v-else>
      <NoPermissionPlaceholder
        v-if="placeholder === 'PERMISSION_DENIED'"
        class="!border-0"
      />
      <NoDataPlaceholder v-else class="!border-0" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { databaseForSpec, useIssueContext } from "@/components/IssueV1/logic";
import NoDataPlaceholder from "@/components/misc/NoDataPlaceholder.vue";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { useCurrentUserV1 } from "@/store";
import { hasProjectPermissionV2 } from "@/utils";
import ReviewOptionSection from "./ReviewOptionSection";

const { isCreating, issue, selectedSpec } = useIssueContext();
const me = useCurrentUserV1();

const database = computed(() => {
  return databaseForSpec(issue.value, selectedSpec.value);
});

const placeholder = computed(() => {
  if (
    isCreating.value &&
    !hasProjectPermissionV2(issue.value.projectEntity, me.value, "bb.plans.get")
  ) {
    return "PERMISSION_DENIED";
  }
  return "NO_DATA";
});
</script>
