<template>
  <div class="flex flex-col gap-y-2 -mt-4">
    <IssueSearch
      v-model:params="state.params"
      :components="['status', 'time-range']"
      :component-props="
        statusTabDisabled ? { status: { disabled: true } } : undefined
      "
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
            mode="PROJECT"
            :show-placeholder="!loading"
            :issue-list="issueList"
          />
        </template>
      </PagedIssueTableV1>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { SearchIcon } from "lucide-vue-next";
import { reactive, PropType, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { TabFilterItem } from "@/components/v2";
import { featureToRef } from "@/store";
import { Project } from "@/types/proto/v1/project_service";
import {
  SearchParams,
  buildIssueFilterBySearchParams,
  buildSearchTextBySearchParams,
  buildUIIssueFilterBySearchParams,
  extractProjectResourceName,
  maybeApplyDefaultTsRange,
  upsertScope,
} from "@/utils";
import { IssueSearch } from "../IssueV1/components";

const TABS = ["WAITING_APPROVAL", "WAITING_ROLLOUT", "ALL"] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  params: SearchParams;
  isFetchingActivityList: boolean;
  loading: boolean;
  loadingMore: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [
      { id: "status", value: "OPEN" },
      { id: "project", value: extractProjectResourceName(props.project.name) },
    ],
  };
  maybeApplyDefaultTsRange(params, "created", true /* mutate */);
  return params;
};

const state = reactive<LocalState>({
  params: defaultSearchParams(),
  isFetchingActivityList: false,
  loading: false,
  loadingMore: false,
});
const { t } = useI18n();

const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");
const keyForTab = (tab: TabValue) => {
  if (tab === "WAITING_APPROVAL") return "project-issues-waiting-approval";
  if (tab === "WAITING_ROLLOUT") return "project-issues-waiting-rollout";
  if (tab === "ALL") return "project-issues-all";

  return "project-issues-unknown";
};

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const items: TabFilterItem<TabValue>[] = [];
  if (hasCustomApprovalFeature.value) {
    items.push({
      value: "WAITING_APPROVAL",
      label: t("issue.waiting-approval"),
    });
  }
  items.push(
    { value: "WAITING_ROLLOUT", label: t("issue.waiting-rollout") },
    { value: "ALL", label: t("common.all") }
  );
  return items;
});
const tab = useLocalStorage<TabValue>(
  "bb.project.issue-list",
  "WAITING_APPROVAL",
  {
    serializer: {
      read(raw: TabValue) {
        if (!TABS.includes(raw)) return "WAITING_APPROVAL";
        return raw;
      },
      write(value) {
        return value;
      },
    },
  }
);
const statusTabDisabled = computed(() => {
  return ["WAITING_APPROVAL", "WAITING_ROLLOUT"].includes(tab.value);
});

const mergeSearchParamsByTab = (tab: TabValue) => {
  const common = cloneDeep(state.params);
  if (tab === "WAITING_APPROVAL") {
    return upsertScope(common, {
      id: "approval",
      value: "pending",
    });
  }
  if (tab === "WAITING_ROLLOUT") {
    return upsertScope(common, {
      id: "approval",
      value: "approved",
    });
  }
  if (tab === "ALL") {
    return common;
  }
  console.error("[mergeSearchParamsByTab] should never reach this line", tab);
  return common;
};

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

watch(
  tab,
  (tab) => {
    if (tab === "WAITING_APPROVAL" || tab === "WAITING_ROLLOUT") {
      upsertScope(
        state.params,
        {
          id: "status",
          value: "OPEN",
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
    if (!hasCustomApprovalFeature.value && tab.value === "WAITING_APPROVAL") {
      tab.value = "WAITING_ROLLOUT";
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
</script>
