<template>
  <div class="flex flex-col gap-y-2 -mt-4">
    <IssueSearch
      v-model:params="state.params"
      :readonly-scopes="readonlyScopes"
      :components="
        state.advanced ? ['searchbox', 'status', 'time-range'] : ['status']
      "
      :component-props="{ status: { disabled: statusTabDisabled } }"
      class="gap-y-1"
    >
      <template v-if="!state.advanced" #default>
        <div class="h-[34px] flex items-center gap-x-2">
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
            mode="PROJECT"
            :show-placeholder="!loading"
            :issue-list="issueList"
            :highlight-text="state.params.query"
          />
        </template>
      </PagedIssueTableV1>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { ChevronDownIcon, SearchIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { reactive, PropType, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { TabFilterItem } from "@/components/v2";
import { Project } from "@/types/proto/v1/project_service";
import {
  SearchParams,
  SearchScope,
  SearchScopeId,
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  buildUIIssueFilterBySearchParams,
  extractProjectResourceName,
  getValueFromSearchParams,
  upsertScope,
} from "@/utils";
import { IssueSearch } from "../IssueV1/components";

const TABS = ["WAITING_APPROVAL", "WAITING_ROLLOUT", "ALL", ""] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  params: SearchParams;
  isFetchingActivityList: boolean;
  advanced: boolean;
  loading: boolean;
  loadingMore: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const readonlyScopes = computed((): SearchScope[] => {
  return [
    { id: "project", value: extractProjectResourceName(props.project.name) },
  ];
});
const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const items: TabFilterItem<TabValue>[] = [
    {
      value: "WAITING_APPROVAL",
      label: t("issue.waiting-approval"),
    },
    { value: "WAITING_ROLLOUT", label: t("issue.waiting-rollout") },
    { value: "ALL", label: t("common.all") },
  ];
  return items;
});
const storedTab = useLocalStorage<TabValue>(
  "bb.project.issue-list-tab",
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
  if (tab === "WAITING_APPROVAL") return "project-issues-waiting-approval";
  if (tab === "WAITING_ROLLOUT") return "project-issues-waiting-rollout";
  if (tab === "ALL") return "project-issues-all";

  return "project-issues-unknown";
};

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [...readonlyScopes.value, { id: "status", value: "OPEN" }],
  };
  return params;
};
const defaultScopeIds = computed(() => {
  return new Set(defaultSearchParams().scopes.map((s) => s.id));
});
const mergeSearchParamsByTab = (params: SearchParams, tab: TabValue) => {
  const common = cloneDeep(params);
  if (tab === "" || tab === "ALL") {
    return common;
  }
  if (tab === "WAITING_APPROVAL") {
    return upsertScope(common, [
      {
        id: "status",
        value: "OPEN",
      },
      {
        id: "approval",
        value: "pending",
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
    ]);
  }
  console.error("[mergeSearchParamsByTab] should never reach this line", tab);
  return common;
};
const guessTabValueFromSearchParams = (params: SearchParams): TabValue => {
  const verifyScopes = (scopes: SearchScopeId[]) => {
    const allowed = new Set(scopes);
    return params.scopes.every(
      (s) => allowed.has(s.id) || defaultScopeIds.value.has(s.id)
    );
  };

  if (
    verifyScopes(["approval"]) &&
    getValueFromSearchParams(params, "status") === "OPEN" &&
    getValueFromSearchParams(params, "approval") === "pending"
  ) {
    return "WAITING_APPROVAL";
  }
  if (
    verifyScopes(["approval"]) &&
    getValueFromSearchParams(params, "status") === "OPEN" &&
    getValueFromSearchParams(params, "approval") === "approved"
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
  isFetchingActivityList: false,
  ...initializeSearchParamsFromQueryOrLocalStorage(),
  loading: false,
  loadingMore: false,
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

const statusTabDisabled = computed(() => {
  if (state.advanced) return false;
  return ["WAITING_APPROVAL", "WAITING_ROLLOUT"].includes(tab.value);
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
  tab,
  () => {
    if (tab.value === "WAITING_APPROVAL" || tab.value === "WAITING_ROLLOUT") {
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
  () => props.project.name,
  () => {
    state.params = defaultSearchParams();
  }
);

watch(
  [() => state.params, tab],
  () => {
    const hash = route.hash;
    if (state.params.query || tab.value === "") {
      // using custom advanced search query, sync the search query string
      // to URL
      router.replace({
        query: {
          ...route.query,
          qs: buildSearchTextBySearchParams(state.params),
        },
        hash,
      });
    } else {
      const query = cloneDeep(route.query);
      delete query["qs"];
      router.replace({
        query,
        hash,
      });
    }
  },
  { deep: true }
);
</script>
