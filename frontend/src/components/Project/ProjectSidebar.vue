<template>
  <nav class="flex-1 flex flex-col overflow-y-hidden">
    <BytebaseLogo class="w-full px-4 shrink-0" />
    <div class="space-y-1 flex-1 overflow-y-auto px-2 pb-4">
      <div v-for="(item, index) in projectSidebarItemList" :key="index">
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
          :class="getItemClass(item.hash)"
          @click="onSelect(item.hash)"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
        </div>
        <div v-if="item.children" class="space-y-1">
          <div
            v-for="child in item.children"
            :key="child.hash"
            class="group w-full flex items-center pl-11 pr-2 py-1.5 rounded-md"
            :class="getItemClass(child.hash)"
            @click="onSelect(child.hash)"
          >
            {{ child.title }}
          </div>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
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
import { useRoute, useRouter } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import { useCurrentUserIamPolicy, useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { idFromSlug } from "@/utils";

const projectHashList = [
  "databases",
  "database-groups",
  "change-history",
  "slow-query",
  "anomaly-center",
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
  hash?: ProjectHash;
  icon: VNode;
  hide?: boolean;
  children?: {
    title: string;
    hash: ProjectHash;
    hide?: boolean;
  }[];
}

interface LocalState {
  selectedHash: ProjectHash;
}

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  selectedHash: "databases",
});

const projectSlug = computed(() => route.params.projectSlug as string);
const project = computed(() => {
  return projectV1Store.getProjectByUID(String(idFromSlug(projectSlug.value)));
});
const currentUserIamPolicy = useCurrentUserIamPolicy();

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const projectSidebarItemList = computed((): ProjectSidebarItem[] => {
  const fullList: ProjectSidebarItem[] = [
    {
      title: t("common.database"),
      icon: h(Database),
      children: [
        {
          title: t("common.databases"),
          hash: "databases",
        },
        {
          title: t("common.groups"),
          hash: "database-groups",
          hide:
            isTenantProject.value ||
            !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: t("common.change-history"),
          hash: "change-history",
          hide:
            isTenantProject.value ||
            !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: startCase(t("slow-query.slow-queries")),
          hash: "slow-query",
          hide: !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
        {
          title: t("common.anomalies"),
          hash: "anomaly-center",
          hide: !currentUserIamPolicy.isMemberOfProject(project.value.name),
        },
      ],
    },
    {
      title: t("common.issues"),
      hash: "issues",
      icon: h(CircleDot),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("common.branches"),
      hash: "branches",
      icon: h(GitBranch),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("changelist.changelists"),
      hash: "changelists",
      icon: h(PencilRuler),
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.allowToChangeDatabaseOfProject(
          project.value.name
        ),
    },
    {
      title: t("database.sync-schema.title"),
      hash: "sync-schema",
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
          hash: "gitops",
        },
        {
          title: t("common.webhooks"),
          hash: "webhook",
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
          hash: "members",
        },
        {
          title: t("common.activities"),
          hash: "activities",
        },
      ],
    },
    {
      title: t("common.setting"),
      icon: h(Settings),
      hash: "setting",
      hide:
        isDefaultProject.value ||
        !currentUserIamPolicy.isMemberOfProject(project.value.name),
    },
  ];

  return fullList
    .map((item) => ({
      ...item,
      children: (item.children ?? []).filter((child) => !child.hide),
    }))
    .filter((item) => !item.hide && (!!item.hash || item.children.length > 0));
});

const getItemClass = (hash: string | undefined) => {
  if (!hash) {
    return [];
  }
  const list = ["outline-item"];
  if (state.selectedHash === hash) {
    list.push("bg-link-hover");
  }
  return list;
};

const onSelect = (hash: ProjectHash | undefined) => {
  if (!hash) {
    return;
  }
  state.selectedHash = hash;
  router.replace({
    name: "workspace.project.detail",
    hash: `#${hash}`,
  });
};

const selectProjectTabOnHash = () => {
  const { name, hash } = router.currentRoute.value;
  if (name == "workspace.project.detail") {
    let targetHash = (hash || "databases").replace(/^#?/g, "");
    if (!isProjectHash(targetHash)) {
      targetHash = "databases";
    }
    onSelect(targetHash as ProjectHash);
  } else if (
    name == "workspace.project.hook.create" ||
    name == "workspace.project.hook.detail"
  ) {
    state.selectedHash = "webhook";
  } else if (name == "workspace.changelist.detail") {
    state.selectedHash = "changelists";
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
