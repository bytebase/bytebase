<template>
  <CommonSidebar
    :key="'project'"
    :item-list="projectSidebarItemList"
    :get-item-class="getItemClass"
  />
</template>

<script setup lang="ts">
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
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
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import { SidebarItem } from "@/components/CommonSidebar.vue";
import {
  PROJECT_V1_WEBHOOK_CREATE,
  PROJECT_V1_WEBHOOK_DETAIL,
  PROJECT_V1_BRANCHE_DETAIL,
  PROJECT_V1_BRANCHE_ROLLOUT,
  PROJECT_V1_BRANCHE_MERGE,
  PROJECT_V1_BRANCHE_REBASE,
  PROJECT_V1_CHANGELIST_DETAIL,
} from "@/router/dashboard/projectV1";
import { useCurrentUserIamPolicy } from "@/store";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { useProjectDatabaseActions } from "../KBar/useDatabaseActions";
import { useCurrentProject } from "./useCurrentProject";

interface ProjectSidebarItem extends SidebarItem {
  title: string;
  type: "div" | "route";
  children?: {
    title: string;
    path: string;
    hide?: boolean;
    type: "route";
  }[];
}

const props = defineProps<{
  projectId?: string;
  issueSlug?: string;
  databaseSlug?: string;
  changeHistorySlug?: string;
}>();

const route = useRoute();
const { t } = useI18n();
const router = useRouter();

const params = computed(() => {
  return {
    projectId: props.projectId,
    issueSlug: props.issueSlug,
    databaseSlug: props.databaseSlug,
    changeHistorySlug: props.changeHistorySlug,
  };
});

const { project } = useCurrentProject(params);

const currentUserIamPolicy = useCurrentUserIamPolicy();

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

// TODO(ed): use route name instead of fullpath
const projectSidebarItemList = computed((): ProjectSidebarItem[] => {
  const projectPath = `/${project.value.name}`;
  return [
    {
      title: t("common.database"),
      icon: h(Database),
      type: "div",
      children: [
        {
          title: t("common.databases"),
          path: `${projectPath}/databases`,
          type: "route",
        },
        {
          title: t("common.groups"),
          path: `${projectPath}/database-groups`,
          type: "route",
          hide:
            !isTenantProject.value ||
            !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: t("common.change-history"),
          path: `${projectPath}/change-histories`,
          type: "route",
          hide:
            isTenantProject.value ||
            !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: startCase(t("slow-query.slow-queries")),
          path: `${projectPath}/slow-queries`,
          type: "route",
          hide: !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: t("common.anomalies"),
          path: `${projectPath}/anomalies`,
          type: "route",
          hide: !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
      ],
    },
    {
      title: t("common.issues"),
      path: `${projectPath}/issues`,
      icon: h(CircleDot),
      type: "route",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
    },
    {
      title: t("common.branches"),
      path: `${projectPath}/branches`,
      icon: h(GitBranch),
      type: "route",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("changelist.changelists"),
      path: `${projectPath}/changelists`,
      icon: h(PencilRuler),
      type: "route",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
    },
    {
      title: t("database.sync-schema.title"),
      path: `${projectPath}/sync-schema`,
      icon: h(RefreshCcw),
      type: "route",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("settings.sidebar.integration"),
      icon: h(Link),
      type: "div",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
      children: [
        {
          title: t("common.gitops"),
          path: `${projectPath}/gitops`,
          type: "route",
        },
        {
          title: t("common.webhooks"),
          path: `${projectPath}/webhooks`,
          type: "route",
        },
      ],
    },
    {
      title: t("common.manage"),
      icon: h(Users),
      type: "div",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
      children: [
        {
          title: t("common.members"),
          path: `${projectPath}/members`,
          type: "route",
        },
        {
          title: t("common.activities"),
          path: `${projectPath}/activities`,
          type: "route",
        },
      ],
    },
    {
      title: t("common.setting"),
      icon: h(Settings),
      path: `${projectPath}/settings`,
      type: "route",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
    },
  ];
});

const getItemClass = (path: string | undefined) => {
  const projectPath = `/${project.value.name}`;

  const list = ["outline-item"];
  switch (route.name) {
    case PROJECT_V1_WEBHOOK_CREATE:
    case PROJECT_V1_WEBHOOK_DETAIL:
      if (path === `${projectPath}/webhooks`) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case PROJECT_V1_BRANCHE_DETAIL:
    case PROJECT_V1_BRANCHE_ROLLOUT:
    case PROJECT_V1_BRANCHE_MERGE:
    case PROJECT_V1_BRANCHE_REBASE:
      if (path === `${projectPath}/branches`) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case PROJECT_V1_CHANGELIST_DETAIL:
      if (path === `${projectPath}/changelists`) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case "workspace.issue.detail":
      if (path === `${projectPath}/issues`) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case "workspace.database.history.detail":
      if (path === `${projectPath}/change-histories`) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
    case "workspace.database.detail":
      if (path === `${projectPath}/databases`) {
        list.push("router-link-active", "bg-link-hover");
      }
      break;
  }
  return list;
};

const onSelect = (path: string) => {
  if (!path) {
    return;
  }
  router.replace({
    path,
  });
};

const flattenNavigationItems = computed(() => {
  return projectSidebarItemList.value.flatMap<{
    path: string;
    title: string;
    hide?: boolean;
  }>((item) => {
    if (item.children && item.children.length > 0) {
      return item.children.map((child) => ({
        path: child.path,
        title: child.title,
        hide: child.hide,
      }));
    }
    return [
      {
        path: item.path ?? "",
        title: item.title,
        hide: item.hide,
      },
    ];
  });
});

const navigationKbarActions = computed(() => {
  const actions = flattenNavigationItems.value.map((item) =>
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
