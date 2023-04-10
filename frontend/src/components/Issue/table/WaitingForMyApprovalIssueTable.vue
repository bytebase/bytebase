<template>
  <slot
    name="table"
    :issue-list="filteredList.map((item) => item.issue)"
    :loading="state.loading"
  />
  <div
    v-if="state.loading"
    class="flex items-center justify-center py-2 text-gray-400 text-sm"
  >
    <BBSpin />
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, watchEffect } from "vue";
import type { Issue, IssueFind } from "@/types";
import { useAuthStore, useIsLoggedIn, useIssueStore } from "@/store";
import {
  extractIssueReviewContext,
  useWrappedReviewSteps,
} from "@/plugins/issue/logic";
import { Review } from "@/types/proto/v1/review_service";

type LocalState = {
  loading: boolean;
  issueList: Issue[];
};

const props = defineProps<{
  issueFind?: IssueFind;
}>();

const state = reactive<LocalState>({
  loading: true,
  issueList: [],
});

const issueStore = useIssueStore();
const isLoggedIn = useIsLoggedIn();
const currentUserName = computed(() => useAuthStore().currentUser.name);

const issueListWithReview = computed(() => {
  return state.issueList.map((issue) => {
    const review = computed(() => {
      try {
        return Review.fromJSON(issue.payload.approval);
      } catch {
        return Review.fromJSON({});
      }
    });
    const context = extractIssueReviewContext(review);
    const steps = useWrappedReviewSteps(issue, context);
    return {
      issue,
      context,
      steps,
    };
  });
});

const filteredList = computed(() => {
  return issueListWithReview.value.filter(({ issue, steps }) => {
    const currentStep = steps.value?.find((step) => step.status === "CURRENT");

    const me = currentStep?.candidates.find(
      (user) => user.name === currentUserName.value
    );

    return me;
  });
});

const fetchData = () => {
  if (!isLoggedIn.value) {
    return;
  }

  state.loading = true;
  issueStore
    .fetchIssueList({
      ...props.issueFind,
      limit: 1000000,
    })
    .then((issueList) => {
      state.issueList = issueList;
    })
    .finally(() => {
      state.loading = false;
    });
};

watchEffect(fetchData);
</script>
