<template>
  <div class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :components="['status', 'time-range']"
      :component-props="{
        status: tab === 'RECENTLY_CLOSED' ? { disabled: true } : undefined,
      }"
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
        :issue-filter="mergeIssueFilterByTab('CREATED')"
        :ui-issue-filter="mergeUIIssueFilterByTab('CREATED')"
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
        :issue-filter="mergeIssueFilterByTab('ASSIGNED')"
        :ui-issue-filter="mergeUIIssueFilterByTab('ASSIGNED')"
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
        :issue-filter="mergeIssueFilterByTab('APPROVAL_REQUESTED')"
        :ui-issue-filter="mergeUIIssueFilterByTab('APPROVAL_REQUESTED')"
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
        :issue-filter="mergeIssueFilterByTab('WAITING_ROLLOUT')"
        :ui-issue-filter="mergeUIIssueFilterByTab('WAITING_ROLLOUT')"
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
        :issue-filter="mergeIssueFilterByTab('SUBSCRIBED')"
        :ui-issue-filter="mergeUIIssueFilterByTab('SUBSCRIBED')"
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

    <div v-show="tab === 'RECENTLY_CLOSED'" class="flex flex-col gap-y-2 pb-2">
      <!-- show the first 5 DONE or CANCELED issues -->
      <!-- But won't show "Load more", since we have a "View all closed" link below -->
      <PagedIssueTableV1
        session-key="home-closed"
        :issue-filter="mergeIssueFilterByTab('RECENTLY_CLOSED')"
        :ui-issue-filter="mergeUIIssueFilterByTab('RECENTLY_CLOSED')"
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
      <div class="w-full flex justify-end px-4">
        <router-link :to="recentlyClosedLink" class="normal-link text-sm">
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
import { cloneDeep } from "lodash-es";
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
import { planTypeToString } from "@/types";
import {
  SearchParams,
  buildIssueFilterBySearchParams,
  buildSearchTextBySearchParams,
  buildUIIssueFilterBySearchParams,
  upsertScope,
} from "@/utils";

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
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const items: TabFilterItem<TabValue>[] = [];
  items.push(
    { value: "CREATED", label: t("common.created") },
    { value: "ASSIGNED", label: t("common.assigned") }
  );
  if (hasCustomApprovalFeature.value) {
    items.push({
      value: "APPROVAL_REQUESTED",
      label: t("issue.approval-requested"),
    });
  }
  items.push(
    { value: "WAITING_ROLLOUT", label: t("issue.waiting-rollout") },
    { value: "SUBSCRIBED", label: t("common.subscribed") },
    { value: "RECENTLY_CLOSED", label: t("project.overview.recently-closed") }
  );
  return items;
});

const mergeSearchParamsByTab = (tab: TabValue) => {
  const common = cloneDeep(state.params);
  const myEmail = me.value.email;
  if (tab === "CREATED") {
    return upsertScope(common, {
      id: "creator",
      value: myEmail,
    });
  }
  if (tab === "ASSIGNED") {
    return upsertScope(common, {
      id: "assignee",
      value: myEmail,
    });
  }
  if (tab === "SUBSCRIBED") {
    return upsertScope(common, {
      id: "subscriber",
      value: myEmail,
    });
  }
  if (tab === "RECENTLY_CLOSED") {
    return upsertScope(common, {
      id: "status",
      value: "CLOSED",
    });
  }
  if (tab === "APPROVAL_REQUESTED") {
    return upsertScope(common, [
      {
        id: "approval",
        value: "pending",
      },
      {
        id: "approver",
        value: myEmail,
      },
    ]);
  }
  if (tab === "WAITING_ROLLOUT") {
    return upsertScope(common, [
      {
        id: "approval",
        value: "approved",
      },
      {
        id: "releaser",
        value: myEmail,
      },
    ]);
  }
  return common;
};

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
  return `/issue?qs=${encodeURIComponent(
    buildSearchTextBySearchParams(mergeSearchParamsByTab(tab.value))
  )}`;
});

const mergeIssueFilterByTab = (tab: TabValue) => {
  return buildIssueFilterBySearchParams(mergeSearchParamsByTab(tab));
};

const mergeUIIssueFilterByTab = (tab: TabValue) => {
  return buildUIIssueFilterBySearchParams(mergeSearchParamsByTab(tab));
};

const recentlyClosedLink = computed(() => {
  return `/issue?qs=${encodeURIComponent(
    buildSearchTextBySearchParams({
      query: "",
      scopes: [{ id: "status", value: "CLOSED" }],
    })
  )}`;
});

watch(
  tab,
  (tab) => {
    if (tab === "RECENTLY_CLOSED") {
      upsertScope(
        state.params,
        {
          id: "status",
          value: "CLOSED",
        },
        true /* mutate */
      );
    }
  },
  { immediate: true }
);

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
