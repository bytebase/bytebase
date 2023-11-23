<template>
  <div class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :components="
        state.advanced ? ['searchbox', 'status', 'time-range'] : ['status']
      "
      :component-props="{ status: { disabled: statusTabDisabled } }"
      class="px-4 py-2 gap-y-1"
    >
      <template v-if="!state.advanced" #default>
        <div class="h-[34px] flex items-center gap-x-2">
          <div class="flex-1 overflow-auto">
            <TabFilter
              v-if="!state.advanced"
              :value="tab"
              :items="tabItemList"
              @update:value="selectTab($event as TabValue)"
            />
          </div>
          <div class="flex items-center">
            <div
              class="flex items-center gap-x-1 normal-link whitespace-nowrap text-sm"
              @click="toggleAdvancedSearch(true)"
            >
              <SearchIcon class="w-4 h-4" />
              <span class="hidden md:block">
                {{ $t("issue.advanced-search.self") }}
              </span>
            </div>
          </div>
        </div>
      </template>
      <template v-if="state.advanced" #searchbox-suffix>
        <NTooltip>
          <template #trigger>
            <NButton
              style="--n-padding: 0 6px; --n-icon-size: 24px"
              @click="toggleAdvancedSearch(false)"
            >
              <template #icon>
                <ChevronDownIcon class="w-5 h-5" />
              </template>
            </NButton>
          </template>
          <template #default>
            <div class="whitespace-nowrap">
              {{ $t("issue.advanced-search.hide") }}
            </div>
          </template>
        </NTooltip>
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
        :issue-filter="mergedIssueFilter"
        :ui-issue-filter="mergedUIIssueFilter"
        :page-size="50"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
            :highlight-text="state.params.query"
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
import { ChevronDownIcon, SearchIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { IssueSearch } from "@/components/IssueV1/components";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { TabFilter, TabFilterItem } from "@/components/v2";
import {
  useSubscriptionV1Store,
  useOnboardingStateStore,
  useCurrentUserV1,
} from "@/store";
import { planTypeToString } from "@/types";
import {
  SearchParams,
  SearchScopeId,
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  buildUIIssueFilterBySearchParams,
  getValueFromSearchParams,
  upsertScope,
} from "@/utils";

const TABS = [
  "CREATED",
  "ASSIGNED",
  "APPROVAL_REQUESTED",
  "WAITING_ROLLOUT",
  "SUBSCRIBED",
  "ALL",
  "",
] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  loading: boolean;
  loadingMore: boolean;
  params: SearchParams;
  advanced: boolean;
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
  return params;
};

const me = useCurrentUserV1();
const route = useRoute();
const router = useRouter();

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
const storedTab = useLocalStorage<TabValue>(
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
const mergeSearchParamsByTab = (params: SearchParams, tab: TabValue) => {
  const common = cloneDeep(params);
  if (tab === "") return common;

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
        id: "status",
        value: "OPEN",
      },
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
        id: "status",
        value: "OPEN",
      },
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
const guessTabValueFromSearchParams = (params: SearchParams): TabValue => {
  const myEmail = me.value.email;

  const verifyScopes = (
    scopes: SearchScopeId[],
    base: SearchScopeId[] = ["status"]
  ) => {
    const allowed = new Set([...scopes, ...base]);
    return params.scopes.every((s) => allowed.has(s.id));
  };

  if (
    verifyScopes(["creator"]) &&
    getValueFromSearchParams(params, "creator") === myEmail
  ) {
    return "CREATED";
  }
  if (
    verifyScopes(["assignee"]) &&
    getValueFromSearchParams(params, "assignee") === myEmail
  ) {
    return "ASSIGNED";
  }
  if (
    verifyScopes(["subscriber"]) &&
    getValueFromSearchParams(params, "subscriber") === myEmail
  ) {
    return "SUBSCRIBED";
  }
  if (
    verifyScopes(["approval", "approver"]) &&
    getValueFromSearchParams(params, "status") === "OPEN" &&
    getValueFromSearchParams(params, "approval") === "pending" &&
    getValueFromSearchParams(params, "approver") === myEmail
  ) {
    return "APPROVAL_REQUESTED";
  }
  if (
    verifyScopes(["approval", "releaser"]) &&
    getValueFromSearchParams(params, "status") === "OPEN" &&
    getValueFromSearchParams(params, "approval") === "approved" &&
    getValueFromSearchParams(params, "releaser") === myEmail
  ) {
    return "WAITING_ROLLOUT";
  }
  if (params.scopes.length === 0) {
    return "ALL";
  }
  if (params.scopes.length === 1 && params.scopes[0].id === "status") {
    return "ALL";
  }
  return "";
};
const initializeSearchParamsFromQueryOrLocalStorage = () => {
  const { qs } = route.query;
  if (typeof qs === "string" && qs.length > 0) {
    return {
      params: buildSearchParamsBySearchText(qs),
      advanced: true,
    };
  }
  return {
    params: mergeSearchParamsByTab(defaultSearchParams(), storedTab.value),
    advanced: false,
  };
};

const state = reactive<LocalState>({
  loading: false,
  loadingMore: false,
  ...initializeSearchParamsFromQueryOrLocalStorage(),
  showTrialStartModal: false,
});

const tab = computed<TabValue>({
  set(tab) {
    if (tab === "") return;
    const base = cloneDeep(state.params);
    base.scopes = base.scopes.filter((s) => s.id === "status");
    state.params = mergeSearchParamsByTab(base, tab);
  },
  get() {
    return guessTabValueFromSearchParams(state.params);
  },
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

const statusTabDisabled = computed(() => {
  if (state.advanced) return false;
  return ["APPROVAL_REQUESTED", "WAITING_ROLLOUT"].includes(tab.value);
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(state.params);
});
const mergedUIIssueFilter = computed(() => {
  return buildUIIssueFilterBySearchParams(state.params);
});

const selectTab = (target: TabValue) => {
  if (target === "") return;
  storedTab.value = target;
  tab.value = target;
};

const toggleAdvancedSearch = (on: boolean) => {
  state.advanced = on;
  if (!on) {
    if (storedTab.value !== "") {
      selectTab(storedTab.value);
    } else {
      selectTab(tabItemList.value[0].value);
    }
    state.params.query = "";
  }
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

watch(
  [() => state.params, tab],
  () => {
    if (state.params.query || tab.value === "") {
      // using custom advanced search query, sync the search query string
      // to URL
      router.replace({
        query: {
          ...route.query,
          qs: buildSearchTextBySearchParams(state.params),
        },
      });
    } else {
      const query = cloneDeep(route.query);
      delete query["qs"];
      router.replace({
        query,
      });
    }
  },
  { deep: true }
);
</script>
