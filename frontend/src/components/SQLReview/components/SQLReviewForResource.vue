<template>
  <div class="flex flex-col gap-y-2">
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
        <Switch
          v-if="allowEditSQLReviewPolicy"
          v-model:value="enforceSQLReviewPolicy"
          :text="true"
        />
        <span
          class="textlabel normal-link text-accent!"
          @click="onSQLReviewPolicyClick"
        >
          {{ pendingSelectReviewPolicy.name }}
          <NButton
            v-if="allowEditSQLReviewTag"
            quaternary
            size="tiny"
            @click.stop="onReviewPolicyRemove"
          >
            <template #icon>
              <XIcon class="w-4 h-auto" />
            </template>
          </NButton>
        </span>
      </div>
      <NButton
        v-else-if="allowEditSQLReviewTag"
        @click.prevent="showReviewSelectPanel = true"
      >
        {{ $t("sql-review.configure-policy") }}
      </NButton>
      <span v-else class="textinfolabel">
        {{ $t("sql-review.no-policy-set") }}
      </span>
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
import { Switch } from "@/components/v2";
import { t } from "@/plugins/i18n";
import { WORKSPACE_ROUTE_SQL_REVIEW_DETAIL } from "@/router/dashboard/workspaceRoutes";
import { useReviewPolicyByResource, useSQLReviewStore } from "@/store";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import SQLReviewPolicySelectPanel from "./SQLReviewPolicySelectPanel.vue";

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const router = useRouter();
const reviewStore = useSQLReviewStore();
const showReviewSelectPanel = ref<boolean>(false);
const pendingSelectReviewPolicy = ref<SQLReviewPolicy | undefined>(undefined);

const scope = computed(() => {
  if (props.resource.startsWith(projectNamePrefix)) {
    return t("settings.general.workspace.query-data-policy.environment-scope");
  }
  if (props.resource.startsWith(environmentNamePrefix)) {
    return t("settings.general.workspace.query-data-policy.project-scope");
  }
  return "";
});

const tooltip = computed(() => {
  if (!scope.value) {
    return "";
  }
  return t("sql-review.tooltip-for-resource", { scope: scope.value });
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

const allowEditSQLReviewPolicy = computed(() => {
  return props.allowEdit && hasWorkspacePermissionV2("bb.reviewConfigs.update");
});

const allowEditSQLReviewTag = computed(() => {
  return props.allowEdit && hasWorkspacePermissionV2("bb.policies.update");
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
