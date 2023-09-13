<template>
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <div><!-- empty --></div>
      <SearchBox
        :value="state.searchText"
        :placeholder="$t('issue.search-issue-name')"
        :autofocus="true"
        @update:value="changeSearchText($event)"
      />
    </div>

    <WaitingForMyApprovalIssueTableV1
      v-if="hasCustomApprovalFeature"
      session-key="home-waiting-approval"
    >
      <template #table="{ issueList, loading }">
        <IssueTableV1
          :show-placeholder="!loading"
          :title="$t('issue.waiting-approval')"
          :issue-list="issueList.filter(keywordFilter)"
          :highlight-text="state.searchText"
        />
      </template>
    </WaitingForMyApprovalIssueTableV1>

    <!-- show OPEN Assigned issues with pageSize=10 -->
    <PagedIssueTableV1
      method="LIST"
      session-key="home-assigned"
      :issue-filter="{
        ...commonIssueFilter,
        statusList: [IssueStatus.OPEN],
        assignee: `users/${currentUserV1.email}`,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTableV1
          class="-mt-px"
          :show-placeholder="!loading"
          :title="$t('issue.waiting-rollout')"
          :issue-list="issueList.filter(keywordFilter)"
          :highlight-text="state.searchText"
        />
      </template>
    </PagedIssueTableV1>

    <!-- show OPEN Created issues with pageSize=10 -->
    <PagedIssueTableV1
      session-key="home-created"
      method="LIST"
      :issue-filter="{
        ...commonIssueFilter,
        statusList: [IssueStatus.OPEN],
        creator: `users/${currentUserV1.email}`,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTableV1
          class="-mt-px"
          :show-placeholder="!loading"
          :title="$t('common.created')"
          :issue-list="issueList.filter(keywordFilter)"
          :highlight-text="state.searchText"
        />
      </template>
    </PagedIssueTableV1>

    <!-- show OPEN Subscribed issues with pageSize=10 -->
    <PagedIssueTableV1
      session-key="home-subscribed"
      method="LIST"
      :issue-filter="{
        ...commonIssueFilter,
        statusList: [IssueStatus.OPEN],
        subscriber: `users/${currentUserV1.email}`,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTableV1
          class="-mt-px"
          :show-placeholder="!loading"
          :title="$t('common.subscribed')"
          :issue-list="issueList.filter(keywordFilter)"
          :highlight-text="state.searchText"
        />
      </template>
    </PagedIssueTableV1>

    <!-- show the first 5 DONE or CANCELED issues -->
    <!-- But won't show "Load more", since we have a "View all closed" link below -->
    <PagedIssueTableV1
      session-key="home-closed"
      method="LIST"
      :issue-filter="{
        ...commonIssueFilter,
        statusList: [IssueStatus.DONE, IssueStatus.CANCELED],
        principal: `users/${currentUserV1.email}`,
      }"
      :page-size="MAX_CLOSED_ISSUE"
      :hide-load-more="true"
    >
      <template #table="{ issueList, loading }">
        <IssueTableV1
          class="-mt-px"
          :show-placeholder="!loading"
          :title="$t('project.overview.recently-closed')"
          :issue-list="issueList.filter(keywordFilter)"
          :highlight-text="state.searchText"
        />
      </template>
    </PagedIssueTableV1>
  </div>
  <div class="w-full flex justify-end mt-2 px-4">
    <router-link
      :to="`/issue?status=closed&user=${currentUserUID}`"
      class="normal-link"
    >
      {{ $t("project.overview.view-all-closed") }}
    </router-link>
  </div>

  <BBModal
    v-if="state.showTrialStartModal && subscriptionStore.subscription"
    :title="
      $t('subscription.trial-start-modal.title', {
        plan: $t(
          `subscription.plan.${planTypeToString(
            subscriptionStore.currentPlan
          ).toLowerCase()}.title`
        ),
      })
    "
    @close="onTrialingModalClose"
  >
    <div class="min-w-0 md:min-w-400 max-w-2xl">
      <div class="flex justify-center items-center">
        <img :src="planImage" class="w-56 px-4" />
        <div class="text-lg space-y-2">
          <p>
            <i18n-t keypath="subscription.trial-start-modal.content">
              <template #plan>
                <strong>
                  {{
                    $t(
                      `subscription.plan.${planTypeToString(
                        subscriptionStore.currentPlan
                      ).toLowerCase()}.title`
                    )
                  }}
                </strong>
              </template>
              <template #date>
                <strong>{{ subscriptionStore.expireAt }}</strong>
              </template>
            </i18n-t>
          </p>
          <p>
            <i18n-t keypath="subscription.trial-start-modal.subscription">
              <template #page>
                <router-link
                  to="/setting/subscription"
                  class="normal-link"
                  exact-active-class=""
                >
                  {{ $t("subscription.trial-start-modal.subscription-page") }}
                </router-link>
              </template>
            </i18n-t>
          </p>
        </div>
      </div>
      <div class="flex justify-end space-x-2 pb-4">
        <button
          type="button"
          class="btn-primary"
          @click.prevent="onTrialingModalClose"
        >
          {{ $t("subscription.trial-start-modal.button") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import WaitingForMyApprovalIssueTableV1 from "@/components/IssueV1/components/WaitingForMyApprovalIssueTableV1.vue";
import { SearchBox } from "@/components/v2";
import {
  useSubscriptionV1Store,
  useOnboardingStateStore,
  featureToRef,
  useCurrentUserV1,
} from "@/store";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { ComposedIssue, IssueFilter, planTypeToString } from "../types";
import { extractUserUID } from "../utils";

interface LocalState {
  searchText: string;
  showTrialStartModal: boolean;
}

const OPEN_ISSUE_LIST_PAGE_SIZE = 10;
const MAX_CLOSED_ISSUE = 5;

const subscriptionStore = useSubscriptionV1Store();
const onboardingStateStore = useOnboardingStateStore();

const state = reactive<LocalState>({
  searchText: "",
  showTrialStartModal: false,
});

const currentUserV1 = useCurrentUserV1();
const currentUserUID = computed(() => extractUserUID(currentUserV1.value.name));
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const onTrialingModalClose = () => {
  state.showTrialStartModal = false;
  onboardingStateStore.consume("show-trialing-modal");
};

const planImage = computed(() => {
  return new URL(
    `../assets/plan-${planTypeToString(
      subscriptionStore.currentPlan
    ).toLowerCase()}.png`,
    import.meta.url
  ).href;
});

const commonIssueFilter = computed((): IssueFilter => {
  return {
    project: "projects/-",
    query: "",
  };
});

const keywordFilter = (issue: ComposedIssue) => {
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    if (!issue.title.toLowerCase().includes(keyword)) {
      return false;
    }
  }
  return true;
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};
</script>
