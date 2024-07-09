<template>
  <div>
    <div class="hidden md:grid">
      <NDataTable
        :striped="true"
        :columns="columns"
        :data="reviewList"
        :row-key="(review: SQLReviewPolicy) => review.id"
      />
    </div>

    <div
      class="flex flex-col md:hidden border px-2 pb-4 divide-y space-y-4 divide-block-border"
    >
      <div
        v-for="(review, i) in reviewList"
        :key="`${i}-${review.id}`"
        class="pt-4"
      >
        <div>
          <span v-if="review.id" class="text-md">
            {{ review.name }}
          </span>
          <span v-else class="italic textinfo text-gray-400">
            {{ $t("sql-review.no-policy-set") }}
          </span>
        </div>
        <div class="space-y-2 space-x-2">
          <BBBadge
            v-for="resource in review.resources"
            :key="resource"
            :can-remove="false"
          >
            <SQLReviewAttachedResource
              :show-prefix="true"
              :resource="resource"
            />
          </BBBadge>
          <BBBadge
            v-if="review.id && !review.enforce"
            :text="$t('sql-review.disabled')"
            :can-remove="false"
            :badge-style="'DISABLED'"
          />
        </div>
        <div class="flex items-center gap-x-2 mt-4">
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

<script setup lang="tsx">
import { NCheckbox, NDataTable, NButton } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBButtonConfirm } from "@/bbkit";
import {
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useCurrentUserV1, useSQLReviewStore } from "@/store";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import { getHighlightHTMLByRegExp } from "@/utils";
import SQLReviewAttachedResource from "./SQLReviewAttachedResource.vue";

const props = defineProps<{
  reviewList: SQLReviewPolicy[];
  filter: string;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const sqlReviewStore = useSQLReviewStore();

const columns = computed((): DataTableColumn<SQLReviewPolicy>[] => {
  return [
    {
      title: t("common.name"),
      key: "name",
      resizable: true,
      render: (review) => {
        return <div innerHTML={highlight(review.name)}></div>;
      },
    },
    {
      title: t("common.resource"),
      key: "resource",
      resizable: true,
      render: (review) => {
        return (
          <div>
            {review.resources.length === 0 && <span>-</span>}
            {review.resources.map((resource) => {
              return (
                <SQLReviewAttachedResource
                  key={resource}
                  resource={resource}
                  showPrefix={true}
                  link={true}
                />
              );
            })}
          </div>
        );
      },
    },
    {
      title: () => {
        return <div class="capitalize">{t("common.enabled")}</div>;
      },
      key: "enabled",
      width: "7rem",
      render: (review) => {
        return (
          <NCheckbox
            disabled={!hasUpdatePolicyPermission.value}
            checked={review.enforce}
            onUpdate:checked={(on) => toggleReviewEnabled(review, on)}
          />
        );
      },
    },
    {
      title: t("common.operations"),
      key: "operations",
      width: "15rem",
      render: (review) => {
        return (
          <div class="flex items-center gap-x-2">
            <NButton onClick={() => handleClickEdit(review)}>
              {hasUpdatePolicyPermission.value
                ? t("common.edit")
                : t("common.view")}
            </NButton>
            {hasDeletePolicyPermission.value && (
              <BBButtonConfirm
                type={"default"}
                style={"DELETE"}
                hideIcon={true}
                buttonText={t("common.delete")}
                okText={t("common.delete")}
                confirmTitle={t("common.delete") + ` '${review.name}'?`}
                requireConfirm={true}
                onConfirm={() => handleClickDelete(review)}
              />
            )}
          </div>
        );
      },
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
