<template>
  <div class="w-full flex flex-col divide-y">
    <template v-if="stageList.length > 0">
      <div class="w-full flex flex-row items-center py-2 px-2 sm:px-4">
        <span class="textlabel mr-4">{{ $t("common.database") }}</span>
        <DatabaseInfo />
      </div>
      <div class="w-full py-2 px-2 sm:px-4">
        <ExportOptionSection />
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
import { useIssueContext } from "@/components/IssueV1/logic";
import NoDataPlaceholder from "@/components/misc/NoDataPlaceholder.vue";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { useCurrentUserV1 } from "@/store";
import { hasProjectPermissionV2 } from "@/utils";
import DatabaseInfo from "./DatabaseInfo.vue";
import ExportOptionSection from "./ExportOptionSection";

const { isCreating, issue } = useIssueContext();
const me = useCurrentUserV1();

// For database data export issue, the stageList should always be only 1 stage.
const stageList = computed(() => {
  return issue.value.rolloutEntity.stages;
});

const placeholder = computed(() => {
  if (
    isCreating.value &&
    !hasProjectPermissionV2(
      issue.value.projectEntity,
      me.value,
      "bb.rollouts.preview"
    )
  ) {
    return "PERMISSION_DENIED";
  }
  return "NO_DATA";
});
</script>
