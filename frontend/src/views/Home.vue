<template>
  <div class="flex flex-col">
    <div v-if="false" class="debug text-xs px-4">
      <pre>{{ state.params }}</pre>
      <pre>{{ mergeIssueFilter(tab) }}</pre>
      <pre>{{ mergeUIIssueFilter(tab) }}</pre>
    </div>

    <IssueSearch
      v-model:params="state.params"
      :components="['status', 'time-range']"
      class="px-4 py-2"
    >
      <template #default>
        <div class="flex items-center gap-x-2">
          <div class="flex-1 overflow-auto">
            <TabFilter v-model:value="tab" :items="tabItemList" />
          </div>
          <div class="flex items-center">
            <router-link
              :to="issueLink"
              class="flex items-center gap-x-1 normal-link whitespace-nowrap text-sm"
            >
              <SearchIcon class="w-4 h-4" />
              <span class="hidden md:block">
                {{ $t("issue.advanced-search.self") }}
              </span>
            </router-link>
          </div>
        </div>
      </template>
    </IssueSearch>

    <div v-show="tab === 'CREATED'">
      <PagedIssueTableV1
        session-key="home-created"
        :issue-filter="mergeIssueFilter('CREATED')"
        :ui-issue-filter="mergeUIIssueFilter('CREATED')"
        :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'ASSIGNED'">
      <PagedIssueTableV1
        session-key="home-assigned"
        :issue-filter="mergeIssueFilter('ASSIGNED')"
        :ui-issue-filter="mergeUIIssueFilter('ASSIGNED')"
        :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'APPROVAL_REQUESTED' && hasCustomApprovalFeature">
      <PagedIssueTableV1
        session-key="home-waiting-approval"
        :issue-filter="mergeIssueFilter('APPROVAL_REQUESTED')"
        :ui-issue-filter="mergeUIIssueFilter('APPROVAL_REQUESTED')"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'WAITING_ROLLOUT'">
      <PagedIssueTableV1
        session-key="home-awaiting-rollout"
        :issue-filter="mergeIssueFilter('WAITING_ROLLOUT')"
        :ui-issue-filter="mergeUIIssueFilter('WAITING_ROLLOUT')"
        :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'SUBSCRIBED'">
      <PagedIssueTableV1
        session-key="home-subscribed"
        :issue-filter="mergeIssueFilter('SUBSCRIBED')"
        :ui-issue-filter="mergeUIIssueFilter('SUBSCRIBED')"
        :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'RECENTLY_CLOSED'">
      <!-- show the first 5 DONE or CANCELED issues -->
      <!-- But won't show "Load more", since we have a "View all closed" link below -->
      <PagedIssueTableV1
        session-key="home-closed"
        :issue-filter="mergeIssueFilter('RECENTLY_CLOSED')"
        :ui-issue-filter="mergeUIIssueFilter('RECENTLY_CLOSED')"
        :page-size="MAX_CLOSED_ISSUE"
        :hide-load-more="true"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
          />
        </template>
      </PagedIssueTableV1>
      <div class="w-full flex justify-end mt-2 px-4">
        <router-link
          :to="`/issue?status=closed&user=${myUID}`"
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
import { SearchIcon } from "lucide-vue-next";
import { reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { IssueSearch } from "@/components/IssueV1/components";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { TabFilter, TabFilterItem } from "@/components/v2";
import {
  useSubscriptionV1Store,
  useOnboardingStateStore,
  featureToRef,
  useCurrentUserV1,
} from "@/store";
import { IssueFilter, planTypeToString } from "../types";
import {
  SearchParams,
  UIIssueFilter,
  buildIssueFilterBySearchParams,
  buildUIIssueFilterBySearchParams,
  extractUserUID,
} from "../utils";

const TABS = [
  "CREATED",
  "ASSIGNED",
  "APPROVAL_REQUESTED",
  "WAITING_ROLLOUT",
  "SUBSCRIBED",
  "RECENTLY_CLOSED",
] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  params: SearchParams;
  showTrialStartModal: boolean;
}

const OPEN_ISSUE_LIST_PAGE_SIZE = 50;
const MAX_CLOSED_ISSUE = 50;

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const onboardingStateStore = useOnboardingStateStore();
const tab = useLocalStorage<TabValue>(
  "bb.home.issue-list-tab",
  "APPROVAL_REQUESTED",
  {
    serializer: {
      read(raw: TabValue) {
        if (!TABS.includes(raw)) return "APPROVAL_REQUESTED";
        return raw;
      },
      write(value) {
        return value;
      },
    },
  }
);

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [
      {
        id: "status",
        value: "OPEN",
      },
    ],
  },
  showTrialStartModal: false,
});

const me = useCurrentUserV1();
const myUID = computed(() => extractUserUID(me.value.name));
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const APPROVAL_REQUESTED: TabFilterItem<TabValue> = {
    value: "APPROVAL_REQUESTED",
    label: t("issue.approval-requested"),
  };
  const list = hasCustomApprovalFeature.value ? [APPROVAL_REQUESTED] : [];
  return [
    { value: "CREATED", label: t("common.created") },
    { value: "ASSIGNED", label: t("common.assigned") },
    ...list,
    { value: "WAITING_ROLLOUT", label: t("issue.waiting-rollout") },
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

const issueLink = computed(() => {
  if (tab.value === "CREATED") {
    return "/issue?creator=" + myUID.value;
  }
  if (tab.value === "APPROVAL_REQUESTED") {
    return "/issue?approver=" + myUID.value;
  }
  if (tab.value === "SUBSCRIBED") {
    return "/issue?subscriber=" + myUID.value;
  }

  // TODO(d): use closed filter for WAITING_ROLLOUT and RECENTLY_CLOSED.
  return "/issue";
});

const mergeIssueFilter = (tab: TabValue): IssueFilter => {
  const common = buildIssueFilterBySearchParams(state.params);
  const myUserTag = `users/${me.value.email}`;
  if (tab === "CREATED") {
    return {
      ...common,
      creator: myUserTag,
    };
  }
  if (tab === "ASSIGNED") {
    return {
      ...common,
      assignee: myUserTag,
    };
  }
  if (tab === "SUBSCRIBED") {
    return {
      ...common,
      subscriber: myUserTag,
    };
  }
  return common;
};

const mergeUIIssueFilter = (tab: TabValue): UIIssueFilter => {
  const common = buildUIIssueFilterBySearchParams(state.params);
  const myUserTag = `users/${me.value.email}`;
  if (tab === "APPROVAL_REQUESTED") {
    return {
      ...common,
      approver: myUserTag,
      approval: "pending",
    };
  }
  if (tab === "WAITING_ROLLOUT") {
    return {
      ...common,
      approval: "approved",
      releaser: myUserTag,
    };
  }
  return common;
};

watch(
  [hasCustomApprovalFeature, tab],
  () => {
    if (!hasCustomApprovalFeature.value && tab.value === "APPROVAL_REQUESTED") {
      tab.value = "WAITING_ROLLOUT";
    }
  },
  { immediate: true }
);
</script>
