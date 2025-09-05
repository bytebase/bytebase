<template>
  <div class="w-full flex flex-col divide-y">
    <template v-if="stageList.length > 0">
      <StageSection />
      <TaskListSection />
      <div class="w-full py-2 px-2 sm:px-4">
        <BBAttention
          v-if="policyStore.maximumResultRows !== Number.MAX_VALUE"
          class="w-full mb-2"
          :type="'warning'"
        >
          {{
            $t(
              "settings.general.workspace.maximum-sql-result.rows.limit-warning",
              { count: policyStore.maximumResultRows }
            )
          }}
        </BBAttention>
        <ExportOptionSection />
      </div>
    </template>
    <template v-else>
      <NoPermissionPlaceholder
        v-if="placeholder === 'PERMISSION_DENIED'"
        class="py-6"
      />
      <NEmpty v-else class="py-6" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NEmpty } from "naive-ui";
import { computed } from "vue";
import { BBAttention } from "@/bbkit";
import { useIssueContext } from "@/components/IssueV1/logic";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { useCurrentProjectV1, usePolicyV1Store } from "@/store";
import { hasProjectPermissionV2 } from "@/utils";
import { StageSection, TaskListSection } from "../../components";
import ExportOptionSection from "./ExportOptionSection";

const { isCreating, issue } = useIssueContext();
const { project } = useCurrentProjectV1();
const policyStore = usePolicyV1Store();

// For database data export issue, the stageList should always be only 1 stage.
const stageList = computed(() => {
  return issue.value.rolloutEntity?.stages || [];
});

const placeholder = computed(() => {
  if (
    isCreating.value &&
    !hasProjectPermissionV2(project.value, "bb.rollouts.preview")
  ) {
    return "PERMISSION_DENIED";
  }
  return "NO_DATA";
});
</script>
