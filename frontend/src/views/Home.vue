<template>
  <div class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :components="['status']"
      :component-props="{ status: { disabled: statusTabDisabled } }"
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

    <div class="relative min-h-[20rem]">
      <div
        v-if="state.loading && !state.loadingMore"
        class="absolute inset-0 bg-white/50 pt-[10rem] flex flex-col items-center"
      >
        <BBSpin />
      </div>

      <PagedIssueTableV1
        :key="keyForTab(tab)"
        v-model:loading="state.loading"
        v-model:loading-more="state.loadingMore"
        :session-key="keyForTab(tab)"
        :issue-filter="mergeIssueFilterByTab(tab)"
        :ui-issue-filter="mergeUIIssueFilterByTab(tab)"
        :page-size="50"
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
  getValueFromSearchParams,
  maybeApplyDefaultTsRange,
  upsertScope,
} from "@/utils";

const TABS = [
  "CREATED",
  "ASSIGNED",
  "APPROVAL_REQUESTED",
  "WAITING_ROLLOUT",
  "SUBSCRIBED",
  "ALL",
] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  loading: boolean;
  loadingMore: boolean;
  params: SearchParams;
  showTrialStartModal: boolean;
}

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const onboardingStateStore = useOnboardingStateStore();

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [
      {
        id: "status",
        value: "OPEN",
      },
    ],
  };
  maybeApplyDefaultTsRange(params, "created", true /* mutate */);
  return params;
};

const state = reactive<LocalState>({
  loading: false,
  loadingMore: false,
  params: defaultSearchParams(),
  showTrialStartModal: false,
});

const me = useCurrentUserV1();

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  return [
    { value: "CREATED", label: t("common.created") },
    { value: "ASSIGNED", label: t("common.assigned") },
    {
      value: "APPROVAL_REQUESTED",
      label: t("issue.approval-requested"),
    },
    { value: "WAITING_ROLLOUT", label: t("issue.waiting-rollout") },
    { value: "SUBSCRIBED", label: t("common.subscribed") },
    { value: "ALL", label: t("common.all") },
  ];
});
const tab = useLocalStorage<TabValue>(
  "bb.home.issue-list-tab",
  tabItemList.value[0].value,
  {
    serializer: {
      read(raw: TabValue) {
        if (!TABS.includes(raw)) return tabItemList.value[0].value;
        return raw;
      },
      write(value) {
        return value;
      },
    },
  }
);

const keyForTab = (tab: TabValue) => {
  if (tab === "CREATED") return "my-issues-created";
  if (tab === "ASSIGNED") return "my-issues-assigned";
  if (tab === "APPROVAL_REQUESTED") return "my-issues-approval-requested";
  if (tab === "WAITING_ROLLOUT") return "my-issues-waiting-rollout";
  if (tab === "SUBSCRIBED") return "my-issues-subscribed";
  if (tab === "ALL") return "my-issues-all";

  return "my-issues-unknown";
};
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
  if (tab === "ALL") {
    return common;
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
  console.error("[mergeSearchParamsByTab] should never reach this line", tab);
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

const statusTabDisabled = computed(() => {
  return ["APPROVAL_REQUESTED", "WAITING_ROLLOUT"].includes(tab.value);
});

const mergeIssueFilterByTab = (tab: TabValue) => {
  return buildIssueFilterBySearchParams(mergeSearchParamsByTab(tab));
};

const mergeUIIssueFilterByTab = (tab: TabValue) => {
  return buildUIIssueFilterBySearchParams(mergeSearchParamsByTab(tab));
};

watch(
  [tab],
  () => {
    if (tab.value === "APPROVAL_REQUESTED" || tab.value === "WAITING_ROLLOUT") {
      if (getValueFromSearchParams(state.params, "status") === "CLOSED") {
        upsertScope(
          state.params,
          {
            id: "status",
            value: "OPEN",
          },
          true /* mutate */
        );
      }
    }
  },
  { immediate: true }
);
</script>
