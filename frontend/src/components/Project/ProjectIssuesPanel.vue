<template>
  <div class="flex flex-col gap-y-2">
    <IssueSearch
      v-model:params="state.params"
      :components="['status', 'time-range']"
      :component-props="{
        status: tab === 'RECENTLY_CLOSED' ? { disabled: true } : undefined,
      }"
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

    <div v-if="hasCustomApprovalFeature" v-show="tab === 'WAITING_APPROVAL'">
      <PagedIssueTableV1
        session-key="project-waiting-approval"
        :issue-filter="mergeIssueFilterByTab('WAITING_APPROVAL')"
        :ui-issue-filter="mergeUIIssueFilterByTab('WAITING_APPROVAL')"
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

    <div v-show="tab === 'WAITING_ROLLOUT'">
      <PagedIssueTableV1
        session-key="project-waiting-rollout"
        :issue-filter="mergeIssueFilterByTab('WAITING_ROLLOUT')"
        :ui-issue-filter="mergeUIIssueFilterByTab('WAITING_ROLLOUT')"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            mode="PROJECT"
            :issue-list="issueList"
            :show-placeholder="!loading"
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="tab === 'RECENTLY_CLOSED'" class="flex flex-col gap-y-2 pb-2">
      <!-- Won't show "Load more", since we have a "View all closed" link below -->
      <PagedIssueTableV1
        session-key="project-closed"
        :issue-filter="mergeIssueFilterByTab('WAITING_ROLLOUT')"
        :ui-issue-filter="mergeUIIssueFilterByTab('WAITING_ROLLOUT')"
        :hide-load-more="true"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            mode="PROJECT"
            :issue-list="issueList"
            :show-placeholder="!loading"
          />
        </template>
      </PagedIssueTableV1>

      <div class="w-full flex justify-end">
        <router-link :to="recentlyClosedLink" class="normal-link text-sm">
          {{ $t("project.overview.view-all-closed") }}
        </router-link>
      </div>
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
  upsertScope,
} from "@/utils";
import { IssueSearch } from "../IssueV1/components";

const TABS = [
  "WAITING_APPROVAL",
  "WAITING_ROLLOUT",
  "RECENTLY_CLOSED",
] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  params: SearchParams;
  isFetchingActivityList: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [
      { id: "project", value: extractProjectResourceName(props.project.name) },
    ],
  },
  isFetchingActivityList: false,
});
const { t } = useI18n();

const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

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
    { value: "RECENTLY_CLOSED", label: t("project.overview.recently-closed") }
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
  if (tab === "RECENTLY_CLOSED") {
    return upsertScope(common, {
      id: "status",
      value: "CLOSED",
    });
  }
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

const recentlyClosedLink = computed(() => {
  return `/issue?qs=${encodeURIComponent(
    buildSearchTextBySearchParams({
      query: "",
      scopes: [
        {
          id: "project",
          value: extractProjectResourceName(props.project.name),
        },
        { id: "status", value: "CLOSED" },
      ],
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
    if (!hasCustomApprovalFeature.value && tab.value === "WAITING_APPROVAL") {
      tab.value = "WAITING_ROLLOUT";
    }
  },
  { immediate: true }
);
</script>
