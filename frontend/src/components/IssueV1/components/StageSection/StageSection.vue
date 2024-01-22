<template>
  <div class="w-full flex flex-col divide-y">
    <template v-if="stageList.length > 0">
      <div class="flex flex-col lg:flex-row justify-between px-4">
        <template v-for="(stage, index) in stageList" :key="stage.uid">
          <StageCard :stage="stage" :index="index" class="h-[54px]" />
          <div
            v-if="index < stageList.length - 1"
            class="hidden lg:block w-3.5 h-[54px] mr-2 pointer-events-none shrink-0"
            aria-hidden="true"
          >
            <svg
              class="h-full w-full text-gray-300"
              viewBox="0 0 22 80"
              fill="none"
              preserveAspectRatio="none"
            >
              <path
                d="M0 -2L20 40L0 82"
                vector-effect="non-scaling-stroke"
                stroke="currentcolor"
                stroke-linejoin="round"
              />
            </svg>
          </div>
        </template>
      </div>

      <StageInfo class="px-4 py-2" />
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
import StageCard from "./StageCard.vue";
import StageInfo from "./StageInfo";

const { isCreating, issue } = useIssueContext();
const me = useCurrentUserV1();

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
