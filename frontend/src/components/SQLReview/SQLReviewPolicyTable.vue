<template>
  <BBGrid
    :column-list="columnList"
    :data-source="combinedList"
    :row-clickable="false"
    class="border"
  >
    <template
      #item="{
        item: { environment, review },
      }: {
        item: EnvironmentReviewPolicy,
      }"
    >
      <div class="bb-grid-cell">
        {{ environment.name }}
        <ProductionEnvironmentIcon :environment="environment" />
      </div>
      <div class="bb-grid-cell">
        <template v-if="review">
          {{ review.name }}
        </template>
      </div>
      <div class="bb-grid-cell justify-center">
        <BBCheckbox
          :disabled="!review || !hasPermission"
          :value="review?.rowStatus === 'NORMAL'"
          @toggle="toggleReviewEnabled(review!, $event)"
        />
      </div>
      <div class="bb-grid-cell gap-x-[14px] !pr-[3rem]">
        <template v-if="!review">
          <button
            type="button"
            class="btn-normal flex justify-center !py-1 !px-3 w-full"
            :disabled="!hasPermission"
            @click.prevent="handleClickCreate(environment)"
          >
            {{ $t("sql-review.configure-policy") }}
          </button>
        </template>
        <template v-else>
          <button
            type="button"
            class="btn-normal flex justify-center !py-1 !px-3"
            @click.prevent="handleClickEdit(review)"
          >
            {{ hasPermission ? $t("common.edit") : $t("common.view") }}
          </button>

          <BBButtonConfirm
            class="btn-normal flex justify-center !py-1 !px-3"
            :disabled="!hasPermission"
            :style="'DELETE'"
            :hide-icon="true"
            :button-text="$t('common.delete')"
            :ok-text="$t('common.delete')"
            :confirm-title="$t('common.delete') + ` '${review.name}'?`"
            :require-confirm="true"
            @confirm="handleClickDelete(review)"
          />
        </template>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import { BBButtonConfirm, BBCheckbox, BBGrid, BBGridColumn } from "@/bbkit";
import {
  pushNotification,
  useCurrentUser,
  useEnvironmentList,
  useSQLReviewStore,
} from "@/store";
import { hasWorkspacePermission, sqlReviewPolicySlug } from "@/utils";
import { Environment, SQLReviewPolicy } from "@/types";
import ProductionEnvironmentIcon from "../Environment/ProductionEnvironmentIcon.vue";

type EnvironmentReviewPolicy = {
  environment: Environment;
  review: SQLReviewPolicy | undefined;
};

const { t } = useI18n();
const router = useRouter();
const currentUser = useCurrentUser();
const sqlReviewStore = useSQLReviewStore();

onMounted(() => {
  sqlReviewStore.fetchReviewPolicyList();
});

const columnList = computed((): BBGridColumn[] => {
  return [
    {
      title: t("common.environment"),
      width: "minmax(auto, 1fr)",
      class: "capitalize",
    },
    {
      title: t("common.name"),
      width: "minmax(auto, 2fr)",
      class: "capitalize",
    },
    {
      title: t("common.enabled"),
      width: "10rem",
      class: "capitalize justify-center",
    },
    {
      title: t("common.operations"),
      width: "auto",
      class: "capitalize",
    },
  ];
});

const hasPermission = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sql-review-policy",
    currentUser.value.role
  );
});

const environmentList = useEnvironmentList();
const reviewPolicyList = computed(() => sqlReviewStore.reviewPolicyList);

const combinedList = computed(() => {
  return environmentList.value.map<EnvironmentReviewPolicy>((environment) => {
    const review = reviewPolicyList.value.find(
      (review) => review.environment.id === environment.id
    );
    return {
      environment,
      review,
    };
  });
});

const toggleReviewEnabled = async (review: SQLReviewPolicy, on: boolean) => {
  await sqlReviewStore.updateReviewPolicy({
    id: review.id,
    rowStatus: on ? "NORMAL" : "ARCHIVED",
  });
};

const handleClickCreate = (environment: Environment) => {
  if (hasPermission.value) {
    router.push({
      name: "setting.workspace.sql-review.create",
      query: {
        environmentId: environment.id,
      },
    });
  } else {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-review.no-permission"),
    });
  }
};

const handleClickEdit = (review: SQLReviewPolicy) => {
  router.push({
    name: "setting.workspace.sql-review.detail",
    params: {
      sqlReviewPolicySlug: sqlReviewPolicySlug(review),
    },
  });
};

const handleClickDelete = async (review: SQLReviewPolicy) => {
  await sqlReviewStore.removeReviewPolicy(review.id);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-review.policy-removed"),
  });
};
</script>
