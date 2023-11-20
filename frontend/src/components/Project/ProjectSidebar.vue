<template>
  <CommonSidebar
    type="div"
    :item-list="projectSidebarItemList"
    :get-item-class="getItemClass"
    @select="(val: string | undefined) => onSelect(val as ProjectHash)"
  />
</template>

<script setup lang="ts">
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
import { useRouter } from "vue-router";
import { SidebarItem } from "@/components/CommonSidebar.vue";
import { useInitializeIssue } from "@/components/IssueV1";
import { useCurrentUserIamPolicy, useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_V1_NAME, unknownProject } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { idFromSlug, projectSlugV1 } from "@/utils";

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
}>();

const issueSlug = computed(() => props.issueSlug ?? "");
const { issue } = useInitializeIssue(issueSlug, false);

const cachedLastPage = useLocalStorage<ProjectHash>(
  `bb.project.${props.projectSlug}.page`,
  "databases"
);

const defaultHash = computed((): ProjectHash => {
  return cachedLastPage.value;
});

interface LocalState {
  selectedHash: ProjectHash;
}

const { t } = useI18n();
const router = useRouter();
const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  selectedHash: defaultHash.value,
});

watch(
  () => state.selectedHash,
  (hash) => (cachedLastPage.value = hash)
);

const project = computed(() => {
  if (props.issueSlug) {
    return issue.value.projectEntity;
  } else if (props.projectSlug) {
    return projectV1Store.getProjectByUID(
      String(idFromSlug(props.projectSlug))
    );
  }
  return unknownProject();
});
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
    params: {
      projectSlug: props.projectSlug || projectSlugV1(project.value),
    },
  });
};

const selectProjectTabOnHash = () => {
  const { name, hash } = router.currentRoute.value;
  if (name == "workspace.project.detail") {
    let targetHash = hash.replace(/^#?/g, "");
    if (!isProjectHash(targetHash)) {
      targetHash = defaultHash.value;
    }
    onSelect(targetHash as ProjectHash);
  } else if (
    name == "workspace.project.hook.create" ||
    name == "workspace.project.hook.detail"
  ) {
    state.selectedHash = "webhook";
  } else if (name == "workspace.changelist.detail") {
    state.selectedHash = "changelists";
  } else if (name === "workspace.branch.detail") {
    state.selectedHash = "branches";
  } else if (
    name === "workspace.database-group.detail" ||
    name === "workspace.database-group.table-group.detail"
  ) {
    state.selectedHash = "database-groups";
  } else if (name === "workspace.issue.detail") {
    state.selectedHash = "issues";
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
</script>
