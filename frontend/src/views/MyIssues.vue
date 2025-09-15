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
            :show-project="true"
          />
        </template>
      </PagedTable>
    </div>
  </div>
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
import { IssueSearch } from "@/components/IssueV1/components";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import type { TabFilterItem } from "@/components/v2";
import { TabFilter } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { WORKSPACE_ROUTE_MY_ISSUES } from "@/router/dashboard/workspaceRoutes";
import {
  useCurrentUserV1,
  useIssueV1Store,
  useRefreshIssueList,
} from "@/store";
import { type ComposedIssue } from "@/types";
import type { SearchParams, SearchScopeId, SemanticIssueStatus } from "@/utils";
import {
  buildIssueFilterBySearchParams,
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
  "ALL",
  "",
] as const;

type TabValue = (typeof TABS)[number];

interface LocalState {
  params: SearchParams;
  advanced: boolean;
}

const { t } = useI18n();
const me = useCurrentUserV1();
const route = useRoute();
const router = useRouter();
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
    },
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
    verifyScopes(["approval", "approver"]) &&
    getSemanticIssueStatusFromSearchParams(params) === "OPEN" &&
    getValueFromSearchParams(params, "approval") === "pending" &&
    getValueFromSearchParams(params, "approver") === myEmail
  ) {
    return "WAITING_APPROVAL";
  }
  if (
    verifyScopes(["approval", "releaser"]) &&
    getSemanticIssueStatusFromSearchParams(params) === "OPEN" &&
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
  return {
    params: mergeSearchParamsByTab(defaultSearchParams(), storedTab.value),
    advanced: false,
  };
};

const state = reactive<LocalState>({
  ...initializeSearchParamsFromQueryOrLocalStorage(),
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
      if (getSemanticIssueStatusFromSearchParams(state.params) === "CLOSED") {
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
  () => tab.value,
  () => {
    if (tab.value !== "") {
      // using custom advanced search query, sync the search query string
      // to URL
      const query = cloneDeep(route.query);
      delete query["qs"];
      router.replace({
        query,
      });
    }
  }
);

watch(
  () => getSemanticIssueStatusFromSearchParams(state.params),
  (status) => {
    storedStatus.value = status;
  }
);
</script>
