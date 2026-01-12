<template>
  <div v-if="allowGetSQLReviewPolicy" class="flex flex-col gap-y-2">
    <div class="flex items-center gap-x-2">
      <label class="font-medium">
        {{ $t("sql-review.title") }}
      </label>
      <NTooltip v-if="tooltip">
        <template #trigger>
          <CircleQuestionMarkIcon class="w-4 textinfolabel" />
        </template>
        <span>
          {{ tooltip }}
        </span>
      </NTooltip>
    </div>
    <div>
      <div
        v-if="pendingSelectReviewPolicy"
        class="inline-flex items-center gap-x-2"
      >
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="[
            'bb.reviewConfigs.update',
          ]"
        >
          <Switch
            v-model:value="enforceSQLReviewPolicy"
            :text="true"
            :disabled="slotProps.disabled"
          />
        </PermissionGuardWrapper>
        <div
          class="flex items-center space-x-1"
        >
          <span class="textlabel normal-link text-accent!" @click="onSQLReviewPolicyClick">
            {{ pendingSelectReviewPolicy.name }}
          </span>
          <NButton
            v-if="hasUpdatePolicyPermission"
            quaternary
            size="tiny"
            @click.stop="onReviewPolicyRemove"
          >
            <template #icon>
              <XIcon class="w-4 h-auto" />
            </template>
          </NButton>
        </div>
      </div>
      <PermissionGuardWrapper
        v-else
        v-slot="slotProps"
        :project="project"
        :permissions="[
          'bb.policies.update',
          'bb.reviewConfigs.list'
        ]"
      >
        <NButton
          :disabled="slotProps.disabled"
          @click.prevent="showReviewSelectPanel = true"
        >
          {{ $t("sql-review.configure-policy") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>
  </div>

  <SQLReviewPolicySelectPanel
    :resource="resource"
    :show="showReviewSelectPanel"
    @close="showReviewSelectPanel = false"
    @select="onReviewPolicySelect"
  />
</template>

<script setup lang="ts">
import { isEqual } from "lodash-es";
import { CircleQuestionMarkIcon, XIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useRouter } from "vue-router";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Switch } from "@/components/v2";
import { t } from "@/plugins/i18n";
import { WORKSPACE_ROUTE_SQL_REVIEW_DETAIL } from "@/router/dashboard/workspaceRoutes";
import {
  useProjectV1Store,
  useReviewPolicyByResource,
  useSQLReviewStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { SQLReviewPolicy } from "@/types";
import { isValidProjectName } from "@/types";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  sqlReviewPolicySlug,
} from "@/utils";
import SQLReviewPolicySelectPanel from "./SQLReviewPolicySelectPanel.vue";

const props = defineProps<{
  resource: string;
}>();

const router = useRouter();
const reviewStore = useSQLReviewStore();
const showReviewSelectPanel = ref<boolean>(false);
const pendingSelectReviewPolicy = ref<SQLReviewPolicy | undefined>(undefined);
const projectStore = useProjectV1Store();

const project = computed(() => {
  if (props.resource.startsWith(projectNamePrefix)) {
    const proj = projectStore.getProjectByName(props.resource);
    if (!isValidProjectName(proj.name)) {
      return undefined;
    }
    return proj;
  }
  return undefined;
});

const hasGetPolicyPermission = computed(() => {
  if (project.value) {
    return hasProjectPermissionV2(project.value, "bb.policies.get");
  }
  return hasWorkspacePermissionV2("bb.policies.get");
});

const hasUpdatePolicyPermission = computed(() => {
  if (project.value) {
    return hasProjectPermissionV2(project.value, "bb.policies.update");
  }
  return hasWorkspacePermissionV2("bb.policies.update");
});

const tooltip = computed(() => {
  if (project.value) {
    return t("sql-review.tooltip-for-resource", {
      scope: t(
        "settings.general.workspace.query-data-policy.environment-scope"
      ),
    });
  }
  return t("sql-review.tooltip-for-resource", {
    scope: t("settings.general.workspace.query-data-policy.project-scope"),
  });
});

const sqlReviewPolicy = useReviewPolicyByResource(
  computed(() => props.resource)
);
const enforceSQLReviewPolicy = ref<boolean>(false);

const resetState = () => {
  pendingSelectReviewPolicy.value = sqlReviewPolicy.value;
  enforceSQLReviewPolicy.value = sqlReviewPolicy.value?.enforce ?? false;
};

watchEffect(resetState);

const allowGetSQLReviewPolicy = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.reviewConfigs.get") &&
    hasGetPolicyPermission.value
  );
});

const onReviewPolicySelect = (review: SQLReviewPolicy) => {
  pendingSelectReviewPolicy.value = review;
  enforceSQLReviewPolicy.value = true;
  showReviewSelectPanel.value = false;
};

const onReviewPolicyRemove = () => {
  pendingSelectReviewPolicy.value = undefined;
  enforceSQLReviewPolicy.value = false;
};

const toggleSQLReviewPolicy = async () => {
  if (!isEqual(sqlReviewPolicy.value, pendingSelectReviewPolicy.value)) {
    if (sqlReviewPolicy.value) {
      // remove resource from old review policy
      await reviewStore.upsertReviewConfigTag({
        oldResources: [...sqlReviewPolicy.value.resources],
        newResources: sqlReviewPolicy.value.resources.filter(
          (resource) => resource !== props.resource
        ),
        review: sqlReviewPolicy.value.id,
      });
    }
    if (pendingSelectReviewPolicy.value) {
      // attach resource to new selected review policy.
      await reviewStore.upsertReviewConfigTag({
        oldResources: [...pendingSelectReviewPolicy.value.resources],
        newResources: [
          ...pendingSelectReviewPolicy.value.resources,
          props.resource,
        ],
        review: pendingSelectReviewPolicy.value.id,
      });
    }
  }

  if (
    pendingSelectReviewPolicy.value &&
    pendingSelectReviewPolicy.value.enforce !== enforceSQLReviewPolicy.value
  ) {
    await reviewStore.upsertReviewPolicy({
      id: pendingSelectReviewPolicy.value.id,
      enforce: enforceSQLReviewPolicy.value,
    });
  }
};

const onSQLReviewPolicyClick = () => {
  if (pendingSelectReviewPolicy.value) {
    router.push({
      name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
      params: {
        sqlReviewPolicySlug: sqlReviewPolicySlug(
          pendingSelectReviewPolicy.value
        ),
      },
    });
  }
};

defineExpose({
  isDirty: computed(
    () =>
      enforceSQLReviewPolicy.value !==
        (pendingSelectReviewPolicy.value?.enforce ?? false) ||
      !isEqual(pendingSelectReviewPolicy.value, sqlReviewPolicy.value)
  ),
  update: toggleSQLReviewPolicy,
  revert: resetState,
});
</script>
