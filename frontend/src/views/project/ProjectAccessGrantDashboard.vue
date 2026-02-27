<template>
  <div class="py-4 w-full flex flex-col">
    <FeatureAttention class="mx-4 mb-2" :feature="PlanFeature.FEATURE_JIT" />

    <ComponentPermissionGuard
      :project="project"
      class="mx-4"
      :permissions="['bb.accessGrants.list']"
    >
      <div class="px-4 pb-2 flex items-center gap-x-2">
        <AdvancedSearch
          class="flex-1"
          v-model:params="searchParams"
          :scope-options="scopeOptions"
          :placeholder="$t('issue.advanced-search.filter')"
          :autofocus="false"
          :cache-query="true"
        />
        <TimeRange
          v-model:show="showTimeRange"
          v-model:params="searchParams"
          @update:params="$emit('update:params', $event)"
        />
      </div>

      <PagedTable
        ref="pagedTableRef"
        :key="projectName"
        :session-key="`project-${projectName}-access-grants`"
        :footer-class="'mx-4'"
        :fetch-list="fetchList"
        :order-keys="ORDER_KEYS"
      >
        <template #table="{ list, loading, sorters, onSortersUpdate }">
          <NDataTable
            v-if="hasJITFeature"
            :columns="getSortedColumns(sorters)"
            :data="list"
            :bordered="false"
            :striped="true"
            :loading="loading"
            :row-key="(row: AccessGrant) => row.name"
            @update:sorter="onSortersUpdate"
          />
          <NEmpty v-else class="py-12 border rounded-sm" />
        </template>
      </PagedTable>
    </ComponentPermissionGuard>

    <BBAttention v-if="!canList" type="info" class="mx-4 mt-2">
      <i18n-t keypath="sql-editor.access-grants-redirect-hint" tag="span">
        <template #link>
          <router-link
            class="normal-link"
            :to="{
              name: 'sql-editor.project',
              params: { project: projectId },
              query: { panel: 'access' },
            }"
          >
            {{ $t("sql-editor.self") }}
          </router-link>
        </template>
      </i18n-t>
    </BBAttention>
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumn, DataTableSortState } from "naive-ui";
import {
  NButton,
  NDataTable,
  NEmpty,
  NTag,
  NTooltip,
  useDialog,
} from "naive-ui";
import { computed, h, ref, type VNode, type VNodeChild, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import AdvancedSearch, { TimeRange } from "@/components/AdvancedSearch";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { FeatureAttention } from "@/components/FeatureGuard";
import YouTag from "@/components/misc/YouTag.vue";
import ComponentPermissionGuard from "@/components/Permission/ComponentPermissionGuard.vue";
import { RichDatabaseName } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { mapSorterStatus } from "@/components/v2/Model/utils";
import {
  type AccessFilter,
  featureToRef,
  pushNotification,
  useAccessGrantStore,
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectByName,
  useUserStore,
} from "@/store";
import { extractUserEmail, projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type AccessGrantFilterStatus,
  extractDatabaseResourceName,
  getAccessGrantDisplayStatus,
  getAccessGrantDisplayStatusText,
  getAccessGrantExpirationText,
  getAccessGrantStatusTagType,
  getDefaultPagination,
  getTsRangeFromSearchParams,
  hasProjectPermissionV2,
  type SearchParams,
} from "@/utils";
import {
  getValueFromSearchParams,
  getValuesFromSearchParams,
} from "@/utils/v1/advanced-search/common";

const PAGE_SIZE = getDefaultPagination();
const ORDER_KEYS = ["creator", "create_time", "expire_time"];

const props = defineProps<{
  projectId: string;
}>();

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const me = useCurrentUserV1();
const userStore = useUserStore();
const databaseStore = useDatabaseV1Store();
const accessGrantStore = useAccessGrantStore();
const showTimeRange = ref(false);

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
const { project } = useProjectByName(projectName);

const pagedTableRef = ref<ComponentExposed<typeof PagedTable<AccessGrant>>>();

const hasJITFeature = featureToRef(PlanFeature.FEATURE_JIT);
const canList = computed(() =>
  hasProjectPermissionV2(project.value, "bb.accessGrants.list")
);
const canActivate = computed(() =>
  hasProjectPermissionV2(project.value, "bb.accessGrants.activate")
);
const canRevoke = computed(() =>
  hasProjectPermissionV2(project.value, "bb.accessGrants.revoke")
);

const searchParams = ref<SearchParams>({
  query: "",
  scopes: [],
});

const scopeOptions = computed((): ScopeOption[] => [
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
  {
    id: "creator",
    title: t("common.creator"),
    search: ({ keyword, nextPageToken: pageToken }) =>
      userStore
        .fetchUserList({
          pageToken,
          pageSize: PAGE_SIZE,
          filter: { query: keyword },
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.users.map<ValueOption>((user) => ({
            value: user.email,
            keywords: [user.email, user.title],
            render: () => {
              const children = [
                h(BBAvatar, { size: "TINY", username: user.title }),
                h("span", user.title),
              ];
              if (user.name === me.value.name) {
                children.push(h(YouTag));
              }
              return h("div", { class: "flex items-center gap-x-1" }, children);
            },
          })),
        })),
  },
  {
    id: "database",
    title: t("common.database"),
    search: ({ keyword, nextPageToken: pageToken }) =>
      databaseStore
        .fetchDatabases({
          parent: projectName.value,
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
  },
]);

const filter = computed((): AccessFilter => {
  const f: AccessFilter = {};
  const statuses = getValuesFromSearchParams(
    searchParams.value,
    "status"
  ) as AccessGrantFilterStatus[];
  f.status = statuses;

  const creator = getValueFromSearchParams(
    searchParams.value,
    "creator",
    undefined
  );
  if (creator) {
    f.creator = `users/${creator}`;
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

  const createdTsRange = getTsRangeFromSearchParams(
    searchParams.value,
    "created"
  );
  if (createdTsRange) {
    f.createdTsAfter = createdTsRange[0];
    f.createdTsBefore = createdTsRange[1];
  }
  return f;
});

const fetchList = async ({
  pageToken,
  pageSize,
  orderBy,
}: {
  pageToken: string;
  pageSize: number;
  orderBy?: string;
}) => {
  const response = await accessGrantStore.listAccessGrants({
    parent: projectName.value,
    filter: filter.value,
    pageSize,
    pageToken: pageToken || undefined,
    orderBy: orderBy ?? "",
  });
  return {
    nextPageToken: response.nextPageToken,
    list: response.accessGrants,
  };
};

watch(
  () => JSON.stringify(filter.value),
  () => pagedTableRef.value?.refresh()
);

const getDatabaseNames = (targets: string[]) => {
  return targets.map((target) => {
    const match = target.match(/databases\/(.+)$/);
    return match ? match[1] : target;
  });
};

const handleActivate = (grant: AccessGrant) => {
  dialog.info({
    title: t("sql-editor.activate-access"),
    content: t("sql-editor.activate-confirm"),
    negativeText: t("common.cancel"),
    positiveText: t("sql-editor.activate-access"),
    onPositiveClick: async () => {
      await accessGrantStore.activateAccessGrant(grant.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.activated"),
      });
      pagedTableRef.value?.refresh();
    },
  });
};

const handleRevoke = (grant: AccessGrant) => {
  dialog.warning({
    title: t("sql-editor.revoke-access"),
    content: t("sql-editor.revoke-confirm"),
    negativeText: t("common.cancel"),
    positiveText: t("sql-editor.revoke-access"),
    onPositiveClick: async () => {
      await accessGrantStore.revokeAccessGrant(grant.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.revoked"),
      });
      pagedTableRef.value?.refresh();
    },
  });
};

const columns = computed((): DataTableColumn<AccessGrant>[] => [
  {
    key: "status",
    title: t("common.status"),
    width: 100,
    render: (grant) => {
      const status = getAccessGrantDisplayStatus(grant);
      return h(
        NTag,
        {
          type: getAccessGrantStatusTagType(status),
          size: "small",
          bordered: false,
        },
        { default: () => getAccessGrantDisplayStatusText(grant) }
      );
    },
  },
  {
    key: "creator",
    title: t("common.creator"),
    width: 180,
    ellipsis: true,
    render: (grant) => extractUserEmail(grant.creator),
  },
  {
    key: "create_time",
    title: t("common.created-at"),
    width: 180,
    render: (grant) => {
      if (!grant.createTime) return "-";
      const ms = getTimeForPbTimestampProtoEs(grant.createTime);
      return h("span", { class: "text-sm" }, new Date(ms).toLocaleString());
    },
  },
  {
    key: "expire_time",
    title: t("common.expiration"),
    sortOrder: false,
    width: 180,
    render: (grant) => {
      const info = getAccessGrantExpirationText(grant);
      if (info.type !== "datetime") return "-";
      return h("span", { class: "text-sm" }, info.value);
    },
  },
  {
    key: "query",
    title: t("common.statement"),
    ellipsis: true,
    render: (grant) =>
      h(
        NTooltip,
        {},
        {
          trigger: () => h("span", { class: "font-mono text-xs" }, grant.query),
          default: () =>
            h("pre", { class: "max-w-lg whitespace-pre-wrap" }, grant.query),
        }
      ),
  },
  {
    key: "targets",
    title: t("common.databases"),
    width: 200,
    render: (grant) => {
      const targets = grant.targets;
      if (targets.length === 0) return "-";
      const renderLink = (target: string): VNodeChild => {
        const name = getDatabaseNames([target])[0];
        return h(
          "span",
          {
            class: "normal-link hover:underline cursor-pointer text-sm",
            onClick: () => router.push({ path: `/${target}` }),
          },
          name
        );
      };
      const visible = targets.slice(0, 2);
      const rest = targets.length - visible.length;
      const children: VNodeChild[] = [];
      visible.forEach((t, i) => {
        if (i > 0) children.push(h("span", { class: "text-sm" }, ", "));
        children.push(renderLink(t));
      });
      if (rest > 0) {
        children.push(
          h("span", { class: "text-sm text-control-placeholder" }, ` +${rest}`)
        );
      }
      const inline = h(
        "div",
        { class: "flex items-center truncate gap-x-0.5" },
        children
      );
      if (rest <= 0) return inline;
      return h(
        NTooltip,
        { style: "background: white; color: inherit" },
        {
          trigger: () => inline,
          default: () =>
            h(
              "div",
              { class: "flex flex-col gap-y-1" },
              targets.map((t) => renderLink(t))
            ),
        }
      );
    },
  },
  {
    key: "actions",
    title: "",
    width: 160,
    render: (grant) => {
      const buttons: VNode[] = [];
      const status = getAccessGrantDisplayStatus(grant);
      if (status === "REVOKED" && canActivate.value) {
        buttons.push(
          h(
            NButton,
            {
              tertiary: true,
              size: "tiny",
              type: "primary",
              onClick: () => handleActivate(grant),
            },
            () => t("sql-editor.activate-access")
          )
        );
      }
      if (status === "ACTIVE" && canRevoke.value) {
        buttons.push(
          h(
            NButton,
            {
              tertiary: true,
              size: "tiny",
              type: "error",
              onClick: () => handleRevoke(grant),
            },
            () => t("sql-editor.revoke-access")
          )
        );
      }
      if (grant.issue) {
        const issuePath = grant.issue.startsWith("/")
          ? grant.issue
          : `/${grant.issue}`;
        buttons.push(
          h(
            "a",
            {
              href: issuePath,
              target: "_blank",
              onClick: (e: Event) => e.stopPropagation(),
            },
            h(NButton, { tertiary: true, size: "tiny" }, () =>
              t("sql-editor.view-issue")
            )
          )
        );
      }
      return h(
        "div",
        { class: "flex items-center justify-end gap-x-1" },
        buttons
      );
    },
  },
]);

const getSortedColumns = (sorters: DataTableSortState[]) =>
  mapSorterStatus(columns.value, sorters);
</script>
