<template>
  <div class="relative w-full h-full flex flex-col justify-start items-start gap-y-1">
    <div
      ref="containerRef"
      class="w-full px-1 flex flex-wrap items-center gap-x-2 gap-y-2"
    >
      <AdvancedSearch
        class="flex-1"
        :class="useSmallLayout ? 'min-w-full' : ''"
        :size="'small'"
        :params="searchParams"
        :scope-options="scopeOptions"
        :placeholder="$t('issue.advanced-search.filter')"
        :autofocus="false"
        :cache-query="false"
        @update:params="searchParams = $event"
      />
      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="[
          'bb.accessGrants.create'
        ]"
        class="ml-auto"
      >
        <NButton
          ghost
          size="small"
          type="primary"
          :style="{
            width: useSmallLayout ? '100%' : 'auto'
          }"
          :disabled="!hasJITFeature || slotProps.disabled"
          @click="showDrawer = true"
        >
          <template v-if="!hasJITFeature" #icon>
            <FeatureBadge :clickable="false" :feature="PlanFeature.FEATURE_JIT" />
          </template>
          {{ $t("sql-editor.request-access") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>
    <div class="w-full flex flex-col justify-start items-start overflow-y-auto">
      <AccessGrantItem
        v-for="grant in accessGrantList"
        :key="grant.name"
        :grant="grant"
        :highlight="grant.name === highlightAccessGrantName"
        :issue="issueByGrantName.get(grant.name)"
        @run="handleRun"
        @request="handleRequest"
      />
      <div
        v-if="nextPageToken"
        class="w-full flex flex-col items-center my-2"
      >
        <NButton
          quaternary
          size="small"
          :loading="loading"
          @click="fetchAccessGrants(false)"
        >
          <span class="textinfolabel">
            {{ $t("common.load-more") }}
          </span>
        </NButton>
      </div>
    </div>

    <template v-if="accessGrantList.length === 0">
      <MaskSpinner v-if="loading" class="bg-white/75!" />
      <div
        v-else
        class="w-full flex items-center justify-center py-8 textinfolabel"
      >
        {{ $t("sql-editor.no-access-requests") }}
      </div>
    </template>

    <AccessGrantRequestDrawer
      v-if="showDrawer"
      :query="pendingCreate?.query"
      :unmask="pendingCreate?.unmask"
      :targets="pendingCreate?.targets"
      @close="handleDrawerClose"
    />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { NButton } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import { FeatureBadge } from "@/components/FeatureGuard";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import {
  type AccessFilter,
  hasFeature,
  useAccessGrantStore,
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useIssueV1Store,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type AccessGrantFilterStatus,
  getAccessSearchOptions,
  getDefaultPagination,
  type SearchParams,
} from "@/utils";
import {
  getValueFromSearchParams,
  getValuesFromSearchParams,
} from "@/utils/v1/advanced-search/common";
import { useSQLEditorContext } from "../../context";
import AccessGrantItem from "./AccessGrantItem.vue";
import AccessGrantRequestDrawer from "./AccessGrantRequestDrawer.vue";

const PAGE_SIZE = getDefaultPagination();

const projectStore = useProjectV1Store();
const editorStore = useSQLEditorStore();
const tabStore = useSQLEditorTabStore();
const accessGrantStore = useAccessGrantStore();
const issueStore = useIssueV1Store();
const { instance: currentInstance } = useConnectionOfCurrentSQLEditorTab();
const { execute } = useExecuteSQL();
const { highlightAccessGrantName } = useSQLEditorContext();

const showDrawer = ref(false);
const loading = ref(false);
const pendingCreate = ref<AccessGrant>();
const accessGrantList = ref<AccessGrant[]>([]);
const nextPageToken = ref("");
// Maps access grant name → Issue for PENDING grants with an associated issue.
const issueByGrantName = ref<Map<string, Issue>>(new Map());
const containerRef = ref<HTMLDivElement>();
const { width: containerWidth } = useElementSize(containerRef);
const useSmallLayout = computed(
  () => containerWidth.value > 0 && containerWidth.value < 250
);

watch(
  () => showDrawer.value,
  (showDrawer) => {
    if (!showDrawer) {
      pendingCreate.value = undefined;
    }
  }
);

const hasJITFeature = computed(() => hasFeature(PlanFeature.FEATURE_JIT));

const project = computed(() =>
  projectStore.getProjectByName(editorStore.project)
);

const searchParams = ref<SearchParams>({
  query: "",
  scopes: [
    {
      id: "status",
      value: AccessGrant_Status[AccessGrant_Status.ACTIVE],
    },
    {
      id: "status",
      value: AccessGrant_Status[AccessGrant_Status.PENDING],
    },
  ],
});

const scopeOptions = computed(() => {
  return getAccessSearchOptions({
    project: editorStore.project,
    showCreator: false,
  });
});

const selectedStatuses = computed(
  () =>
    getValuesFromSearchParams(
      searchParams.value,
      "status"
    ) as AccessGrantFilterStatus[]
);

// Build AccessFilter from search params.
const filter = computed((): AccessFilter => {
  const f: AccessFilter = {
    status: selectedStatuses.value,
  };

  const database = getValueFromSearchParams(
    searchParams.value,
    "database",
    undefined
  );
  if (database) {
    f.target = database;
  }

  const queryText = searchParams.value.query.trim();
  if (queryText) {
    f.statement = queryText;
  }

  return f;
});

const fetchIssuesForPendingGrants = async (grants: AccessGrant[]) => {
  const pendingWithIssue = grants.filter(
    (g) => g.status === AccessGrant_Status.PENDING && g.issue
  );
  const results = await Promise.all(
    pendingWithIssue.map(async (g) => {
      try {
        const issue = await issueStore.fetchIssueByName(g.issue, true);
        return { grantName: g.name, issue };
      } catch {
        return undefined;
      }
    })
  );
  for (const r of results) {
    if (r) {
      issueByGrantName.value.set(r.grantName, r.issue);
    }
  }
};

const fetchAccessGrants = async (resetList = true) => {
  const project = editorStore.project;
  if (!project) return;

  loading.value = true;
  try {
    const response = await accessGrantStore.searchMyAccessGrants({
      parent: project,
      filter: filter.value,
      pageSize: PAGE_SIZE,
      pageToken: resetList ? undefined : nextPageToken.value,
    });
    if (resetList) {
      accessGrantList.value = response.accessGrants;
      issueByGrantName.value = new Map();
    } else {
      accessGrantList.value.push(...response.accessGrants);
    }
    nextPageToken.value = response.nextPageToken;
    await fetchIssuesForPendingGrants(response.accessGrants);
  } finally {
    loading.value = false;
  }
};

// Re-fetch when project or filter changes.
watch([() => editorStore.project, filter], () => fetchAccessGrants(), {
  immediate: true,
  deep: true,
});

// When a new grant is highlighted (e.g. created from the drawer in other panels),
// refresh the list so it appears, then auto-clear the highlight after 3 seconds.
watch(
  highlightAccessGrantName,
  async (name) => {
    if (name) {
      await fetchAccessGrants();
      setTimeout(() => {
        if (highlightAccessGrantName.value === name) {
          highlightAccessGrantName.value = undefined;
        }
      }, 3000);
    }
  },
  { immediate: true }
);

const handleDrawerClose = () => {
  showDrawer.value = false;
  fetchAccessGrants();
};

const handleRequest = async (grant: AccessGrant) => {
  pendingCreate.value = grant;
  showDrawer.value = true;
};

const handleRun = async (grant: AccessGrant) => {
  const database = grant.targets[0] ?? "";
  const instanceName = database.replace(/\/databases\/.*$/, "");
  // Pre-fetch the database so the connection is fully resolved
  // before the tab is created and the query is executed.
  await useDatabaseV1Store().getOrFetchDatabaseByName(database);
  const tab = tabStore.addTab(
    {
      connection: { instance: instanceName, database },
      statement: grant.query,
      batchQueryContext: {
        databases: grant.targets,
      },
    },
    /* beside */ true
  );
  nextTick(() => {
    execute({
      connection: { ...tab.connection },
      statement: grant.query,
      engine: currentInstance.value.engine,
      explain: false,
      selection: null,
    });
  });
};
</script>
