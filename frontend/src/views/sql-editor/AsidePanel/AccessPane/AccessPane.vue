<template>
  <div class="relative w-full h-full flex flex-col justify-start items-start gap-y-1">
    <div
      ref="containerRef"
      class="w-full px-1 flex flex-wrap items-center gap-x-2 gap-y-1"
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
      <NButton
        ghost
        size="small"
        type="primary"
        :style="{
          width: useSmallLayout ? '100%' : 'auto'
        }"
        :disabled="!hasJITFeature"
        @click="showDrawer = true"
      >
        <template v-if="!hasJITFeature" #icon>
          <FeatureBadge :clickable="false" :feature="PlanFeature.FEATURE_JIT" />
        </template>
        {{ $t("sql-editor.request-access") }}
      </NButton>
    </div>
    <div class="w-full flex flex-col justify-start items-start overflow-y-auto">
      <AccessGrantItem
        v-for="grant in accessGrantList"
        :key="grant.name"
        :grant="grant"
        :highlight="grant.name === highlightAccessGrantName"
        @run="handleRun"
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
      @close="handleDrawerClose"
    />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { NButton } from "naive-ui";
import { computed, h, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import AdvancedSearch from "@/components/AdvancedSearch";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { FeatureBadge } from "@/components/FeatureGuard";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { RichDatabaseName } from "@/components/v2";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import {
  type AccessFilter,
  hasFeature,
  useAccessGrantStore,
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  extractDatabaseResourceName,
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

const { t } = useI18n();
const editorStore = useSQLEditorStore();
const tabStore = useSQLEditorTabStore();
const databaseStore = useDatabaseV1Store();
const accessGrantStore = useAccessGrantStore();
const { instance: currentInstance } = useConnectionOfCurrentSQLEditorTab();
const { execute } = useExecuteSQL();
const { highlightAccessGrantName } = useSQLEditorContext();

const showDrawer = ref(false);
const loading = ref(false);
const accessGrantList = ref<AccessGrant[]>([]);
const nextPageToken = ref("");
const containerRef = ref<HTMLDivElement>();
const { width: containerWidth } = useElementSize(containerRef);
const useSmallLayout = computed(
  () => containerWidth.value > 0 && containerWidth.value < 250
);

const hasJITFeature = computed(() => hasFeature(PlanFeature.FEATURE_JIT));

const searchParams = ref<SearchParams>({
  query: "",
  scopes: [],
});

const scopeOptions = computed((): ScopeOption[] => {
  const options: ScopeOption[] = [
    {
      id: "status",
      title: t("common.status"),
      allowMultiple: true,
      options: [
        {
          value: AccessGrant_Status[AccessGrant_Status.ACTIVE],
          keywords: ["active"],
          render: () => t("common.active"),
        },
        {
          value: AccessGrant_Status[AccessGrant_Status.PENDING],
          keywords: ["pending"],
          render: () => t("common.pending"),
        },
        {
          value: "EXPIRED",
          keywords: ["expired"],
          render: () => t("sql-editor.expired"),
        },
        {
          value: AccessGrant_Status[AccessGrant_Status.REVOKED],
          keywords: ["revoked"],
          render: () => t("common.revoked"),
        },
      ],
    },
  ];
  const project = editorStore.project;
  if (project) {
    options.push({
      id: "database",
      title: t("common.database"),
      search: ({ keyword, nextPageToken: pageToken }) =>
        databaseStore
          .fetchDatabases({
            parent: project,
            pageToken: pageToken,
            pageSize: PAGE_SIZE,
            filter: { query: keyword },
          })
          .then((resp) => ({
            nextPageToken: resp.nextPageToken,
            options: resp.databases.map<ValueOption>((db) => {
              const { database: dbName } = extractDatabaseResourceName(db.name);
              return {
                value: db.name,
                keywords: [dbName, db.name],
                render: () =>
                  h(RichDatabaseName, {
                    database: db,
                    showInstance: true,
                    showEngineIcon: true,
                  }),
              };
            }),
          })),
    });
  }
  return options;
});

const selectedStatuses = computed(() =>
  getValuesFromSearchParams(searchParams.value, "status")
);

const statusMap: Record<string, AccessGrant_Status> = {
  ACTIVE: AccessGrant_Status.ACTIVE,
  PENDING: AccessGrant_Status.PENDING,
  REVOKED: AccessGrant_Status.REVOKED,
  EXPIRED: AccessGrant_Status.ACTIVE,
};

// Build AccessFilter from search params.
const filter = computed((): AccessFilter => {
  const f: AccessFilter = {};

  const statuses = selectedStatuses.value;
  if (statuses.length === 1) {
    f.status = statusMap[statuses[0]];
    if (statuses[0] === "EXPIRED") {
      f.expireTsBefore = Date.now();
    } else if (statuses[0] === "ACTIVE") {
      f.expireTsAfter = Date.now();
    }
  }

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
    } else {
      accessGrantList.value.push(...response.accessGrants);
    }
    nextPageToken.value = response.nextPageToken;
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
      accessGrant: grant.name,
    });
  });
};
</script>
