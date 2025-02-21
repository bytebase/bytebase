<template>
  <div class="flex flex-col gap-y-2">
    <label class="textlabel">
      {{ $t("sql-review.title") }}
    </label>
    <div>
      <div v-if="sqlReviewPolicy" class="inline-flex items-center gap-x-2">
        <Switch
          v-if="allowEditSQLReviewPolicy"
          v-model:value="enforceSQLReviewPolicy"
          :text="true"
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
import { useRouter } from "vue-router";
import { Switch } from "@/components/v2";
import { WORKSPACE_ROUTE_SQL_REVIEW_DETAIL } from "@/router/dashboard/workspaceRoutes";
import { useSQLReviewStore, useReviewPolicyByResource } from "@/store";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import SQLReviewPolicySelectPanel from "./SQLReviewPolicySelectPanel.vue";

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const router = useRouter();
const reviewStore = useSQLReviewStore();
const showReviewSelectPanel = ref<boolean>(false);

const sqlReviewPolicy = useReviewPolicyByResource(
  computed(() => props.resource)
);
const enforceSQLReviewPolicy = ref<boolean>(
  sqlReviewPolicy.value?.enforce ?? false
);

const allowEditSQLReviewPolicy = computed(() => {
  return props.allowEdit && hasWorkspacePermissionV2("bb.policies.update");
});

const toggleSQLReviewPolicy = async () => {
  const policy = sqlReviewPolicy.value;
  if (!policy) return;
  const originalOn = policy.enforce;
  if (enforceSQLReviewPolicy.value === originalOn) return;
  await reviewStore.upsertReviewPolicy({
    id: policy.id,
    enforce: enforceSQLReviewPolicy.value,
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

defineExpose({
  isDirty: computed(
    () =>
      enforceSQLReviewPolicy.value !== (sqlReviewPolicy.value?.enforce ?? false)
  ),
  update: toggleSQLReviewPolicy,
  revert: () =>
    (enforceSQLReviewPolicy.value = sqlReviewPolicy.value?.enforce ?? false),
});
</script>
