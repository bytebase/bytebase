<template>
  <div class="flex flex-col">
    <div class="px-2 flex items-center">
      <div class="flex-1 overflow-hidden">
        <TabFilter v-model:value="tab" :items="tabItemList" />
      </div>
      <div class="flex items-center space-x-2 p-0.5">
        <router-link
          :to="`/issue?user=${currentUserUID}`"
          class="flex space-x-1 items-center normal-link !whitespace-nowrap"
        >
          <heroicons-outline:search class="h-4 w-4" />
          <span class="hidden md:block">{{
            $t("issue.advanced-search.self")
          }}</span>
        </router-link>
      </div>
    </div>
    <div v-show="tab === 'REVIEW_REQUESTED'" class="mt-2">
      <PagedIssueTableV1
        v-if="hasCustomApprovalFeature"
        session-key="home-waiting-approval"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.OPEN],
        }"
        :ui-issue-filter="{
          approver: `users/${currentUserV1.email}`,
          status: 'pending_approval',
        }"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'WAITING_ROLLOUT'" class="mt-2">
      <!-- show OPEN Assigned issues with pageSize=10 -->
      <PagedIssueTableV1
        session-key="home-assigned"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.OPEN],
          assignee: `${userNamePrefix}${currentUserV1.email}`,
        }"
        :ui-issue-filter="{
          status: 'pending_rollout',
        }"
        :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'CREATED'" class="mt-2">
      <!-- show OPEN Created issues with pageSize=10 -->
      <PagedIssueTableV1
        session-key="home-created"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.OPEN],
          creator: `${userNamePrefix}${currentUserV1.email}`,
        }"
        :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'SUBSCRIBED'" class="mt-2">
      <!-- show OPEN Subscribed issues with pageSize=10 -->
      <PagedIssueTableV1
        session-key="home-subscribed"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.OPEN],
          subscriber: `${userNamePrefix}${currentUserV1.email}`,
        }"
        :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'RECENTLY_CLOSED'" class="mt-2">
      <!-- show the first 5 DONE or CANCELED issues -->
      <!-- But won't show "Load more", since we have a "View all closed" link below -->
      <PagedIssueTableV1
        session-key="home-closed"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.DONE, IssueStatus.CANCELED],
          principal: `${userNamePrefix}${currentUserV1.email}`,
        }"
        :page-size="MAX_CLOSED_ISSUE"
        :hide-load-more="true"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
            title=""
          />
        </template>
      </PagedIssueTableV1>
      <div class="w-full flex justify-end mt-2 px-4">
        <router-link
          :to="`/issue?status=closed&user=${currentUserUID}`"
          class="normal-link"
        >
          {{ $t("project.overview.view-all-closed") }}
        </router-link>
      </div>
    </div>
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
import { useLocalStorage } from "@vueuse/core";
import { reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { TabFilter, TabFilterItem } from "@/components/v2";
import {
  useSubscriptionV1Store,
  useOnboardingStateStore,
  featureToRef,
  useCurrentUserV1,
} from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { IssueFilter, planTypeToString } from "../types";
import { extractUserUID } from "../utils";

const TABS = [
  "REVIEW_REQUESTED",
  "WAITING_ROLLOUT",
  "CREATED",
  "SUBSCRIBED",
  "RECENTLY_CLOSED",
] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  showTrialStartModal: boolean;
}

const OPEN_ISSUE_LIST_PAGE_SIZE = 50;
const MAX_CLOSED_ISSUE = 50;

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const onboardingStateStore = useOnboardingStateStore();
const tab = useLocalStorage<TabValue>(
  "bb.home.issue-list-tab",
  "REVIEW_REQUESTED",
  {
    serializer: {
      read(raw: TabValue) {
        if (!TABS.includes(raw)) return "REVIEW_REQUESTED";
        return raw;
      },
      write(value) {
        return value;
      },
    },
  }
);

const state = reactive<LocalState>({
  showTrialStartModal: false,
});

const currentUserV1 = useCurrentUserV1();
const currentUserUID = computed(() => extractUserUID(currentUserV1.value.name));
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const REVIEW_REQUESTED: TabFilterItem<TabValue> = {
    value: "REVIEW_REQUESTED",
    label: t("issue.review-requested"),
  };
  const list = hasCustomApprovalFeature.value ? [REVIEW_REQUESTED] : [];
  return [
    ...list,
    { value: "WAITING_ROLLOUT", label: t("issue.waiting-rollout") },
    { value: "CREATED", label: t("common.created") },
    { value: "SUBSCRIBED", label: t("common.subscribed") },
    { value: "RECENTLY_CLOSED", label: t("project.overview.recently-closed") },
  ];
});

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

watch(
  [hasCustomApprovalFeature, tab],
  () => {
    if (!hasCustomApprovalFeature.value && tab.value === "REVIEW_REQUESTED") {
      tab.value = "WAITING_ROLLOUT";
    }
  },
  { immediate: true }
);
</script>
