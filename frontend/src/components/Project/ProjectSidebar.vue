<template>
  <CommonSidebar
    type="div"
    :item-list="projectSidebarItemList"
    :get-item-class="getItemClass"
    @select="(val: string | undefined) => onSelect(val as ProjectHash)"
  />
</template>

<script setup lang="ts">
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useLocalStorage } from "@vueuse/core";
import { startCase } from "lodash-es";
import {
  Database,
  GitBranch,
  CircleDot,
  Users,
  Link,
  Settings,
  RefreshCcw,
  PencilRuler,
} from "lucide-vue-next";
import { computed, VNode, h, reactive, watch, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import { SidebarItem } from "@/components/CommonSidebar.vue";
import { useCurrentUserIamPolicy } from "@/store";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { projectSlugV1 } from "@/utils";
import { useProjectDatabaseActions } from "../KBar/useDatabaseActions";
import { useCurrentProject } from "./useCurrentProject";

const projectHashList = [
  "databases",
  "database-groups",
  "change-history",
  "slow-query",
  "anomalies",
  "issues",
  "branches",
  "changelists",
  "sync-schema",
  "gitops",
  "webhook",
  "members",
  "activities",
  "setting",
] as const;
export type ProjectHash = typeof projectHashList[number];
const isProjectHash = (x: any): x is ProjectHash => projectHashList.includes(x);

interface ProjectSidebarItem {
  title: string;
  path?: ProjectHash;
  icon: VNode;
  hide?: boolean;
  children?: {
    title: string;
    path: ProjectHash;
    hide?: boolean;
  }[];
}

const props = defineProps<{
  projectSlug?: string;
  issueSlug?: string;
  databaseSlug?: string;
  changeHistorySlug?: string;
}>();

const route = useRoute();

interface LocalState {
  selectedHash: ProjectHash;
}

const { t } = useI18n();
const router = useRouter();

const state = reactive<LocalState>({
  selectedHash: "databases",
});

const params = computed(() => {
  return {
    projectSlug: props.projectSlug,
    issueSlug: props.issueSlug,
    databaseSlug: props.databaseSlug,
    changeHistorySlug: props.changeHistorySlug,
  };
});

const { project } = useCurrentProject(params);

const cachedLastPage = computed(() => {
  const cache = useLocalStorage<ProjectHash>(
    `bb.project.${projectSlugV1(project.value)}.page`,
    "databases"
  );
  return cache;
});

const defaultHash = computed((): ProjectHash => {
  return cachedLastPage.value.value;
});

watch(
  () => state.selectedHash,
  (hash) => (cachedLastPage.value.value = hash)
);

watch(
  () => cachedLastPage.value,
  (cache) => (state.selectedHash = cache.value)
);

const currentUserIamPolicy = useCurrentUserIamPolicy();

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const projectSidebarItemList = computed((): SidebarItem[] => {
  const fullList: ProjectSidebarItem[] = [
    {
      title: t("common.database"),
      icon: h(Database),
      children: [
        {
          title: t("common.databases"),
          path: "databases",
        },
        {
          title: t("common.groups"),
          path: "database-groups",
          hide:
            !isTenantProject.value ||
            !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: t("common.change-history"),
          path: "change-history",
          hide:
            isTenantProject.value ||
            !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: startCase(t("slow-query.slow-queries")),
          path: "slow-query",
          hide: !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: t("common.anomalies"),
          path: "anomalies",
          hide: !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
      ],
    },
    {
      title: t("common.issues"),
      path: "issues",
      icon: h(CircleDot),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("common.branches"),
      path: "branches",
      icon: h(GitBranch),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("changelist.changelists"),
      path: "changelists",
      icon: h(PencilRuler),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("database.sync-schema.title"),
      path: "sync-schema",
      icon: h(RefreshCcw),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("settings.sidebar.integration"),
      icon: h(Link),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
      children: [
        {
          title: t("common.gitops"),
          path: "gitops",
        },
        {
          title: t("common.webhooks"),
          path: "webhook",
        },
      ],
    },
    {
      title: t("common.manage"),
      icon: h(Users),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
      children: [
        {
          title: t("common.members"),
          path: "members",
        },
        {
          title: t("common.activities"),
          path: "activities",
        },
      ],
    },
    {
      title: t("common.setting"),
      icon: h(Settings),
      path: "setting",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
    },
  ];

  return fullList as SidebarItem[];
});

const getItemClass = (hash: string | undefined) => {
  if (!hash) {
    return [];
  }
  const list = ["outline-item"];
  if (!isProjectHash(hash)) {
    return list;
  }
  const projectHash = hash as ProjectHash;
  if (state.selectedHash === projectHash) {
    list.push("bg-link-hover");
  }
  return list;
};

const onSelect = (hash: ProjectHash | undefined) => {
  if (!hash || !isProjectHash(hash)) {
    return;
  }
  state.selectedHash = hash;
  router.replace({
    name: "workspace.project.detail",
    hash: `#${hash}`,
    query: route.query,
    params: {
      projectSlug: props.projectSlug || projectSlugV1(project.value),
    },
  });
};

const selectProjectTabOnHash = () => {
  const { name, hash } = router.currentRoute.value;

  switch (name) {
    case "workspace.project.detail": {
      let targetHash = hash.replace(/^#?/g, "");
      if (!isProjectHash(targetHash)) {
        targetHash = defaultHash.value;
      }
      onSelect(targetHash as ProjectHash);
      return;
    }
    case "workspace.project.hook.create":
    case "workspace.project.hook.detail":
      state.selectedHash = "webhook";
      return;
    case "workspace.changelist.detail":
      state.selectedHash = "changelists";
      return;
    case "workspace.branch.detail":
      state.selectedHash = "branches";
      return;
    case "workspace.database-group.detail":
    case "workspace.database-group.table-group.detail":
      state.selectedHash = "database-groups";
      return;
    case "workspace.issue.detail":
      state.selectedHash = "issues";
      return;
    case "workspace.database.detail":
    case "workspace.database.history.detail":
      state.selectedHash = "databases";
      return;
  }
};

watch(
  () => [router.currentRoute.value.hash],
  () => {
    nextTick(() => {
      selectProjectTabOnHash();
    });
  },
  {
    immediate: true,
  }
);

const navigationKbarActions = computed(() => {
  const navigationItems = projectSidebarItemList.value.flatMap<{
    path: ProjectHash;
    title: string;
    hide?: boolean;
  }>((item) => {
    if (item.children && item.children.length > 0) {
      return item.children.map((child) => ({
        path: child.path as ProjectHash,
        title: child.title,
        hide: child.hide,
      }));
    }
    return [
      {
        path: item.path as ProjectHash,
        title: item.title,
        hide: item.hide,
      },
    ];
  });

  const actions = navigationItems.map((item) =>
    defineAction({
      id: `bb.navigation.project.${project.value.uid}.${item.path}`,
      name: item.title,
      section: t("kbar.navigation"),
      keywords: [item.title.toLowerCase(), item.path].join(" "),
      perform: () => onSelect(item.path),
    })
  );
  return actions;
});
useRegisterActions(navigationKbarActions);

useProjectDatabaseActions(project);
</script>
