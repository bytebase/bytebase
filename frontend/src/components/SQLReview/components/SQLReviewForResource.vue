<template>
  <div class="flex flex-col gap-y-2">
    <label class="textlabel">
      {{ $t("sql-review.title") }}
    </label>
    <div>
      <div v-if="sqlReviewPolicy" class="inline-flex items-center gap-x-2">
        <Switch
          v-if="allowEditSQLReviewPolicy"
          :value="sqlReviewPolicy.enforce"
          :text="true"
          @update:value="toggleSQLReviewPolicy"
        />
        <span
          class="textlabel normal-link !text-accent"
          @click="onSQLReviewPolicyClick"
        >
          {{ sqlReviewPolicy.name }}
        </span>
      </div>
      <NButton
        v-else-if="allowEditSQLReviewPolicy"
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
  />
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { Switch } from "@/components/v2";
import { WORKSPACE_ROUTE_SQL_REVIEW_DETAIL } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useSQLReviewStore,
  useReviewPolicyByResource,
} from "@/store";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import SQLReviewPolicySelectPanel from "./SQLReviewPolicySelectPanel.vue";

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const router = useRouter();
const reviewStore = useSQLReviewStore();
const showReviewSelectPanel = ref<boolean>(false);

const allowEditSQLReviewPolicy = computed(() => {
  return props.allowEdit && hasWorkspacePermissionV2("bb.policies.update");
});

const toggleSQLReviewPolicy = async (on: boolean) => {
  const policy = sqlReviewPolicy.value;
  if (!policy) return;
  const originalOn = policy.enforce;
  if (on === originalOn) return;
  await reviewStore.upsertReviewPolicy({
    id: policy.id,
    enforce: on,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-review.policy-updated"),
  });
};

const onSQLReviewPolicyClick = () => {
  if (sqlReviewPolicy.value) {
    router.push({
      name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
      params: {
        sqlReviewPolicySlug: sqlReviewPolicySlug(sqlReviewPolicy.value),
      },
    });
  }
};

const sqlReviewPolicy = useReviewPolicyByResource(
  computed(() => props.resource)
);
</script>
