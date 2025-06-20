<template>
  <NButton text :disabled="isLoading" @click="handleClick">
    <BBSpin v-if="isLoading" :size="20" class="mr-1" />
    <span>
      {{ isSubscribed ? $t("issue.unsubscribe") : $t("issue.subscribe") }}
    </span>
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { BBSpin } from "@/bbkit";
import { useIssueContext, toggleSubscribeIssue } from "@/components/IssueV1";
import { useCurrentUserV1 } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";

const { issue } = useIssueContext();
const currentUser = useCurrentUserV1();
const isLoading = ref(false);

const isSubscribed = computed(() => {
  return issue.value.subscribers.includes(
    `${userNamePrefix}${currentUser.value.email}`
  );
});

const handleClick = async () => {
  try {
    isLoading.value = true;
    await toggleSubscribeIssue(issue.value);
  } finally {
    isLoading.value = false;
  }
};
</script>
