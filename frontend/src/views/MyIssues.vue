<template>
  <div :key="viewId" class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :components="
        state.advanced ? ['searchbox', 'time-range', 'status'] : ['status']
      "
      :component-props="{ status: { hidden: statusTabHidden } }"
      class="px-4 pb-2"
    >
      <template v-if="!state.advanced" #default>
        <div class="flex items-center gap-x-2">
          <div class="flex-1 overflow-auto">
            <TabFilter
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
      <PagedTable
        ref="issuePagedTable"
        :key="keyForTab(tab)"
        :session-key="`bb.issue-table.${keyForTab(tab)}`"
        :fetch-list="fetchIssueList"
      >
        <template #table="{ list, loading }">
          <IssueTableV1
            class="border-x-0"
            :loading="loading"
            :issue-list="applyUIIssueFilter(list, mergedUIIssueFilter)"
            :highlight-text="state.params.query"
          />
        </template>
      </PagedTable>
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
                  :to="{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }"
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
        <NButton type="primary" @click.prevent="onTrialingModalClose">
          {{ $t("subscription.trial-start-modal.button") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useLocalStorage, type UseStorageOptions } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { ChevronDownIcon, SearchIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { reactive, computed, watch, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBModal } from "@/bbkit";
import { IssueSearch } from "@/components/IssueV1/components";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import type { TabFilterItem } from "@/components/v2";
import { TabFilter } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { WORKSPACE_ROUTE_MY_ISSUES } from "@/router/dashboard/workspaceRoutes";
import { SETTING_ROUTE_WORKSPACE_SUBSCRIPTION } from "@/router/dashboard/workspaceSetting";
import {
  useSubscriptionV1Store,
  useOnboardingStateStore,
  useCurrentUserV1,
  useAppFeature,
  useIssueV1Store,
  useRefreshIssueList,
} from "@/store";
import { planTypeToString, type ComposedIssue } from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import type { SearchParams, SearchScopeId, SemanticIssueStatus } from "@/utils";
import {
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  buildUIIssueFilterBySearchParams,
  getSemanticIssueStatusFromSearchParams,
  getValueFromSearchParams,
  upsertScope,
  useDynamicLocalStorage,
  applyUIIssueFilter,
} from "@/utils";
import { getComponentIdLocalStorageKey } from "@/utils/localStorage";

const TABS = [
  "CREATED",
  "WAITING_APPROVAL",
  "WAITING_ROLLOUT",
  "SUBSCRIBED",
  "ALL",
  "",
] as const;

type TabValue = (typeof TABS)[number];

interface LocalState {
  params: SearchParams;
  advanced: boolean;
  showTrialStartModal: boolean;
}

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const onboardingStateStore = useOnboardingStateStore();
const me = useCurrentUserV1();
const route = useRoute();
const router = useRouter();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
const issueStore = useIssueV1Store();
const issuePagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedIssue>>>();

const viewId = useLocalStorage<string>(
  getComponentIdLocalStorageKey(WORKSPACE_ROUTE_MY_ISSUES),
  ""
);

const storedStatus = useDynamicLocalStorage<SemanticIssueStatus>(
  computed(() => `bb.home.issue-list-status.${me.value.name}`),
  "OPEN",
  window.localStorage,
  {
    serializer: {
      read(raw: SemanticIssueStatus) {
        if (!["OPEN", "CLOSED"].includes(raw)) return "OPEN";
        return raw;
      },
      write(value) {
        return value;
      },
    },
  } as UseStorageOptions<SemanticIssueStatus>
);

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [
      {
        id: "status",
        value: storedStatus.value,
      },
    ],
  };
  return params;
};
const defaultScopeIds = computed(() => {
  return new Set(defaultSearchParams().scopes.map((s) => s.id));
});
const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const items: (TabFilterItem<TabValue> & { hide?: boolean })[] = [
    { value: "ALL", label: t("common.all") },
    { value: "CREATED", label: t("common.created") },
    {
      value: "WAITING_APPROVAL",
      label: t("issue.waiting-approval"),
    },
    {
      value: "WAITING_ROLLOUT",
      label: t("issue.waiting-rollout"),
      hide: databaseChangeMode.value === DatabaseChangeMode.EDITOR,
    },
    { value: "SUBSCRIBED", label: t("common.subscribed") },
  ];
  return items.filter((item) => !item.hide);
});
const storedTab = useDynamicLocalStorage<TabValue>(
  computed(() => `bb.home.issue-list-tab.${me.value.name}`),
  tabItemList.value[0].value,
  window.localStorage,
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
  } as UseStorageOptions<TabValue>
);
const keyForTab = (tab: TabValue) => {
  if (tab === "CREATED") return "my-issues-created";
  if (tab === "WAITING_APPROVAL") return "my-issues-waiting-approval";
  if (tab === "WAITING_ROLLOUT") return "my-issues-waiting-rollout";
  if (tab === "SUBSCRIBED") return "my-issues-subscribed";
  if (tab === "ALL") return "my-issues-all";

  return "my-issues-unknown";
};
const mergeSearchParamsByTab = (params: SearchParams, tab: TabValue) => {
  const common = cloneDeep(params);
  if (tab === "" || tab === "ALL") {
    return common;
  }

  const myEmail = me.value.email;
  if (tab === "CREATED") {
    return upsertScope({
      params: common,
      scopes: {
        id: "creator",
        value: myEmail,
      },
    });
  }
  if (tab === "SUBSCRIBED") {
    return upsertScope({
      params: common,
      scopes: {
        id: "subscriber",
        value: myEmail,
      },
    });
  }
  if (tab === "WAITING_APPROVAL") {
    return upsertScope({
      params: common,
      scopes: [
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
      ],
    });
  }
  if (tab === "WAITING_ROLLOUT") {
    return upsertScope({
      params: common,
      scopes: [
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
      ],
    });
  }
  console.error("[mergeSearchParamsByTab] should never reach this line", tab);
  return common;
};
const guessTabValueFromSearchParams = (params: SearchParams): TabValue => {
  const myEmail = me.value.email;

  const verifyScopes = (scopes: SearchScopeId[]) => {
    const allowed = new Set([...scopes]);
    return params.scopes.every(
      (s) => allowed.has(s.id) || defaultScopeIds.value.has(s.id)
    );
  };

  if (
    verifyScopes(["creator"]) &&
    getValueFromSearchParams(params, "creator") === myEmail
  ) {
    return "CREATED";
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
    return "WAITING_APPROVAL";
  }
  if (
    verifyScopes(["approval", "releaser"]) &&
    getValueFromSearchParams(params, "status") === "OPEN" &&
    getValueFromSearchParams(params, "approval") === "approved" &&
    getValueFromSearchParams(params, "releaser") === myEmail
  ) {
    return "WAITING_ROLLOUT";
  }

  if (params.scopes.every((s) => defaultScopeIds.value.has(s.id))) {
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
  ...initializeSearchParamsFromQueryOrLocalStorage(),
  showTrialStartModal: false,
});

const tab = computed<TabValue>({
  set(tab) {
    if (tab === "") return;
    const base = cloneDeep(state.params);
    base.scopes = base.scopes.filter((s) => defaultScopeIds.value.has(s.id));
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
    `@/assets/plan-${planTypeToString(
      subscriptionStore.currentPlan
    ).toLowerCase()}.png`,
    import.meta.url
  ).href;
});

const statusTabHidden = computed(() => {
  if (state.advanced) return true;
  return ["WAITING_APPROVAL", "WAITING_ROLLOUT"].includes(tab.value);
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(state.params);
});
const mergedUIIssueFilter = computed(() => {
  return buildUIIssueFilterBySearchParams(state.params);
});

const fetchIssueList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, issues } = await issueStore.listIssues({
    find: mergedIssueFilter.value,
    pageSize,
    pageToken,
  });
  return {
    nextPageToken,
    list: issues,
  };
};

watch(
  () => JSON.stringify(mergedIssueFilter.value),
  () => issuePagedTable.value?.refresh()
);
useRefreshIssueList(() => issuePagedTable.value?.refresh());

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
    if (tab.value === "WAITING_APPROVAL" || tab.value === "WAITING_ROLLOUT") {
      if (getValueFromSearchParams(state.params, "status") === "CLOSED") {
        upsertScope({
          params: state.params,
          scopes: {
            id: "status",
            value: "OPEN",
          },
          mutate: true,
        });
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

watch(
  () => getSemanticIssueStatusFromSearchParams(state.params),
  (status) => {
    storedStatus.value = status;
  }
);
</script>
