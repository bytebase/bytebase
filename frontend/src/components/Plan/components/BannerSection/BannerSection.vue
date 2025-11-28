<template>
  <div>
    <div
      v-show="showClosedBanner"
      class="h-8 w-full text-base font-medium bg-gray-400 text-white flex justify-center items-center shrink-0"
    >
      {{ $t("common.closed") }}
    </div>
    <div
      v-show="showSuccessBanner"
      class="h-8 w-full text-base font-medium bg-success text-white flex justify-center items-center shrink-0"
    >
      {{ $t("common.done") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { usePlanContext } from "@/components/Plan";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";

const { plan, issue } = usePlanContext();

const showClosedBanner = computed(() => {
  return (
    plan.value.state === State.DELETED ||
    (issue.value && issue.value.status === IssueStatus.CANCELED)
  );
});

const showSuccessBanner = computed(() => {
  return issue.value && issue.value.status === IssueStatus.DONE;
});
</script>
