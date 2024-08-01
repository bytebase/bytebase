<template>
  <div class="flex items-center gap-x-2">
    <SubscribeButton v-if="allowSubscribe" />
    <SubscriberList :readonly="!allowSubscribe" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { useCurrentUserV1 } from "@/store";
import { hasProjectPermissionV2 } from "@/utils";
import SubscribeButton from "./SubscribeButton.vue";
import SubscriberList from "./SubscriberList.vue";

const { issue } = useIssueContext();
const currentUser = useCurrentUserV1();

const allowSubscribe = computed(() => {
  return hasProjectPermissionV2(
    issue.value.projectEntity,
    currentUser.value,
    "bb.issues.update"
  );
});
</script>
