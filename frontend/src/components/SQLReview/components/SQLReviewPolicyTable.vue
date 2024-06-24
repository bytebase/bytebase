<template>
  <div>
    <BBGrid
      :column-list="columnList"
      :data-source="reviewList"
      :row-clickable="false"
      class="border hidden md:grid"
    >
      <template #item="{ item: review }: { item: SQLReviewPolicy }">
        <div class="bb-grid-cell">
          <span v-if="review.resources.length === 0">-</span>
          <div v-else>
            <SQLReviewAttachedResource
              v-for="resource in review.resources"
              :key="resource"
              :resource="resource"
              :show-prefix="true"
              :link="true"
            />
          </div>
        </div>
        <div class="bb-grid-cell">
          <template v-if="review.id">
            <div :innerHTML="highlight(review.name)" />
          </template>
          <span v-else class="italic textinfo text-gray-400">
            {{ $t("sql-review.no-policy-set") }}
          </span>
        </div>
        <div class="bb-grid-cell justify-center">
          <NCheckbox
            :disabled="!review.id || !hasUpdatePolicyPermission"
            :checked="review.enforce ?? false"
            @update:checked="toggleReviewEnabled(review, $event)"
          />
        </div>
        <div class="bb-grid-cell gap-x-2 !pr-[3rem]">
          <template v-if="!review.id">
            <NButton
              :disabled="!hasUpdatePolicyPermission"
              @click.prevent="handleClickCreate(review.resources[0])"
            >
              {{ $t("sql-review.configure-policy") }}
            </NButton>
          </template>
          <template v-else>
            <NButton @click.prevent="handleClickEdit(review)">
              {{
                hasUpdatePolicyPermission
                  ? $t("common.edit")
                  : $t("common.view")
              }}
            </NButton>

            <BBButtonConfirm
              v-if="hasDeletePolicyPermission"
              type="default"
              :disabled="!hasUpdatePolicyPermission"
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

    <div
      class="flex flex-col md:hidden border px-2 pb-4 divide-y space-y-4 divide-block-border"
    >
      <div
        v-for="(review, i) in reviewList"
        :key="`${i}-${review.id}`"
        class="pt-4 space-y-3"
      >
        <div>
          <span v-if="review.id" class="text-md">
            {{ review.name }}
          </span>
          <span v-else class="italic textinfo text-gray-400">
            {{ $t("sql-review.no-policy-set") }}
          </span>
        </div>
        <div class="flex items-center gap-x-2">
          <BBBadge
            v-for="resource in review.resources"
            :key="resource"
            :can-remove="false"
          >
            <SQLReviewAttachedResource :resource="resource" />
          </BBBadge>
          <BBBadge
            v-if="review.id && !review.enforce"
            :text="$t('sql-review.disabled')"
            :can-remove="false"
            :badge-style="'DISABLED'"
            ::badge-style="'DISABLED'"
          />
        </div>
        <div class="flex items-center gap-x-2">
          <template v-if="!review.id">
            <NButton
              :disabled="!hasUpdatePolicyPermission"
              @click.prevent="handleClickCreate(review.resources[0])"
            >
              {{ $t("sql-review.configure-policy") }}
            </NButton>
          </template>
          <template v-else>
            <NButton @click.prevent="handleClickEdit(review)">
              {{
                hasUpdatePolicyPermission
                  ? $t("common.edit")
                  : $t("common.view")
              }}
            </NButton>

            <BBButtonConfirm
              v-if="hasDeletePolicyPermission"
              type="default"
              :disabled="!hasUpdatePolicyPermission"
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
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { BBGridColumn } from "@/bbkit";
import { BBButtonConfirm, BBGrid } from "@/bbkit";
import {
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useCurrentUserV1, useSQLReviewStore } from "@/store";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import { getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  reviewList: SQLReviewPolicy[];
  filter: string;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const sqlReviewStore = useSQLReviewStore();

const columnList = computed((): BBGridColumn[] => {
  return [
    {
      title: t("common.resource"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("common.name"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("common.enabled"),
      width: "minmax(min-content, auto)",
      class: "capitalize justify-center",
    },
    {
      title: t("common.operations"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
  ];
});

const highlight = (content: string) => {
  return getHighlightHTMLByRegExp(
    content,
    props.filter.toLowerCase().split(" "),
    /* !caseSensitive */ false,
    /* className */ "bg-yellow-100"
  );
};

const hasCreatePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.create");
});

const hasUpdatePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update");
});

const hasDeletePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.delete");
});

const handleClickCreate = (attachedResource: string | undefined) => {
  if (hasCreatePolicyPermission.value) {
    router.push({
      name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
      query: {
        attachedResource,
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
    name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
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

const toggleReviewEnabled = async (review: SQLReviewPolicy, on: boolean) => {
  await sqlReviewStore.updateReviewPolicy({
    id: review.id,
    enforce: on,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
