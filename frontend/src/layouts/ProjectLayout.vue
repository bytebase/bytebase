<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.state === State.DELETED" />
  </div>
  <h1 class="px-6 pb-4 text-xl font-bold leading-6 text-main truncate">
    <template v-if="isDefaultProject">
      {{ $t("database.unassigned-databases") }}
    </template>
    <template v-else>
      {{ project.title }}
    </template>
    <span
      v-if="isTenantProject"
      class="text-sm font-normal px-2 ml-2 rounded whitespace-nowrap inline-flex items-center bg-gray-200"
    >
      {{ $t("project.mode.batch") }}
    </span>
  </h1>
  <BBAttention
    v-if="isDefaultProject"
    class="mx-6 mb-4"
    :style="'INFO'"
    :title="$t('project.overview.info-slot-content')"
  />
  <BBTabFilter
    class="px-3 pb-2 border-b border-block-border"
    :responsive="false"
    :tab-item-list="tabItemList"
    :selected-index="state.selectedIndex"
    @select-index="selectTab"
  />

  <div class="py-6 px-6">
    <router-view
      :project-slug="projectSlug"
      :project-webhook-slug="projectWebhookSlug"
      :allow-edit="allowEdit"
    />
  </div>
</template>

<script lang="ts" setup>
import { startCase } from "lodash-es";
import { computed, nextTick, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBTabFilterItem } from "@/bbkit/types";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import {
  useCurrentUserIamPolicy,
  useCurrentUserV1,
  useProjectV1Store,
} from "@/store";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  idFromSlug,
  hasWorkspacePermissionV1,
  hasPermissionInProjectV1,
} from "@/utils";

type ProjectTabItem = {
  name: string;
  hash: string;
};

interface LocalState {
  selectedIndex: number;
}

const props = defineProps({
  projectSlug: {
    required: true,
    type: String,
  },
  projectWebhookSlug: {
    type: String,
    default: undefined,
  },
});

const router = useRouter();
const { t } = useI18n();

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByUID(String(idFromSlug(props.projectSlug)));
});
const currentUserIamPolicy = useCurrentUserIamPolicy();

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const projectTabItemList = computed((): ProjectTabItem[] => {
  if (!currentUserIamPolicy.isMemberOfProject(project.value.name)) {
    return [{ name: t("common.databases"), hash: "databases" }];
  }
  if (
    !currentUserIamPolicy.allowToChangeDatabaseOfProject(project.value.name)
  ) {
    const list = [{ name: t("common.databases"), hash: "databases" }];
    if (!isDefaultProject.value) {
      list.push(
        ...[
          { name: t("common.members"), hash: "members" },
          { name: t("common.settings"), hash: "setting" },
        ]
      );
    }
    return list;
  }

  const list: (ProjectTabItem | null)[] = [
    { name: t("common.overview"), hash: "overview" },
    { name: t("common.databases"), hash: "databases" },

    isTenantProject.value
      ? { name: t("common.database-groups"), hash: "database-groups" }
      : null,

    isTenantProject.value
      ? null // Hide "Change History" tab for tenant projects
      : { name: t("common.change-history"), hash: "change-history" },

    { name: startCase(t("slow-query.slow-queries")), hash: "slow-query" },

    { name: t("common.activities"), hash: "activity" },
    isDefaultProject.value
      ? null
      : { name: t("common.gitops"), hash: "gitops" },
    isDefaultProject.value
      ? null
      : { name: t("common.webhooks"), hash: "webhook" },
    isDefaultProject.value
      ? null
      : { name: t("common.members"), hash: "members" },
    isDefaultProject.value
      ? null
      : { name: t("common.settings"), hash: "setting" },
  ];
  const filteredList = list.filter((item) => item !== null) as ProjectTabItem[];

  return filteredList;
});

const findTabIndexByHash = (hash: string) => {
  hash = hash.replace(/^#?/g, "");
  const index = projectTabItemList.value.findIndex(
    (item) => item.hash === hash
  );
  if (index >= 0) {
    return index;
  }
  // otherwise fallback to the first tab
  return 0;
};

const state = reactive<LocalState>({
  selectedIndex: findTabIndexByHash(router.currentRoute.value.hash),
});

const allowEdit = computed(() => {
  if (project.value.state === State.DELETED) {
    return false;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.manage-general"
    )
  ) {
    return true;
  }
  return false;
});

const tabItemList = computed((): BBTabFilterItem[] => {
  return projectTabItemList.value.map((item) => {
    return {
      title: item.name,
      alert: false,
    };
  });
});

const selectProjectTabOnHash = () => {
  const { name, hash } = router.currentRoute.value;
  if (name == "workspace.project.detail") {
    const index = findTabIndexByHash(hash);
    selectTab(index);
  } else if (
    name == "workspace.project.hook.create" ||
    name == "workspace.project.hook.detail"
  ) {
    state.selectedIndex = findTabIndexByHash("webhook");
  }
};

const selectTab = (index: number) => {
  state.selectedIndex = index;
  router.replace({
    name: "workspace.project.detail",
    hash: "#" + projectTabItemList.value[index].hash,
  });
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
