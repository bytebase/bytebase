<template>
  <NDataTable
    key="sql-review-table"
    :size="size"
    :striped="true"
    :columns="columns"
    :data="reviewList"
    :row-props="rowProps"
    :row-key="(review: SQLReviewPolicy) => review.id"
  />
</template>

<script setup lang="tsx">
import { CheckIcon, XIcon } from "lucide-vue-next";
import { NCheckbox, NDataTable, NButton } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import { pushNotification, useSQLReviewStore } from "@/store";
import type { SQLReviewPolicy } from "@/types";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { getHighlightHTMLByRegExp } from "@/utils";
import SQLReviewAttachedResource from "./SQLReviewAttachedResource.vue";

const props = withDefaults(
  defineProps<{
    size?: "small" | "medium";
    reviewList: SQLReviewPolicy[];
    filter?: string;
    customClick?: boolean;
    allowEdit: boolean;
  }>(),
  {
    size: "medium",
    filter: "",
    customClick: false,
  }
);

const emit = defineEmits<{
  (event: "row-click", review: SQLReviewPolicy): void;
  (event: "edit", review: SQLReviewPolicy): void;
  (event: "delete", review: SQLReviewPolicy): void;
}>();

const { t } = useI18n();
const sqlReviewStore = useSQLReviewStore();

const columns = computed(
  (): (DataTableColumn<SQLReviewPolicy> & { hide?: boolean })[] => {
    return [
      {
        title: t("common.name"),
        key: "name",
        resizable: true,
        render: (review: SQLReviewPolicy) => {
          return <div innerHTML={highlight(review.name)}></div>;
        },
      },
      {
        title: t("common.resource"),
        key: "resource",
        resizable: true,
        render: (review: SQLReviewPolicy) => {
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
        title: t("sql-review.enabled-rules"),
        key: "rules",
        render: (review: SQLReviewPolicy) => {
          return (
            <span>
              {
                review.ruleList.filter(
                  (r) => r.level !== SQLReviewRuleLevel.DISABLED
                ).length
              }
            </span>
          );
        },
      },
      {
        title: () => {
          return <div class="capitalize">{t("common.enabled")}</div>;
        },
        key: "enabled",
        width: "7rem",
        render: (review: SQLReviewPolicy) => {
          return props.allowEdit ? (
            <NCheckbox
              disabled={!hasUpdatePolicyPermission.value || !props.allowEdit}
              checked={review.enforce}
              onUpdate:checked={(on) => toggleReviewEnabled(review, on)}
            />
          ) : review.enforce ? (
            <CheckIcon class="w-4 h-4 textinfolabel" />
          ) : (
            <XIcon class="w-4 h-4 textinfolabel" />
          );
        },
      },
      {
        title: t("common.operations"),
        key: "operations",
        width: "15rem",
        hide: !props.allowEdit,
        render: (review: SQLReviewPolicy) => {
          return (
            <div class="flex items-center gap-x-2">
              <NButton onClick={() => emit("edit", review)}>
                {hasUpdatePolicyPermission.value
                  ? t("common.edit")
                  : t("common.view")}
              </NButton>
              {hasDeletePolicyPermission.value && (
                <BBButtonConfirm
                  text={false}
                  type={"DELETE"}
                  hideIcon={true}
                  buttonText={t("common.delete")}
                  okText={t("common.delete")}
                  confirmTitle={t("common.delete") + ` '${review.name}'?`}
                  requireConfirm={true}
                  onConfirm={() => emit("delete", review)}
                />
              )}
            </div>
          );
        },
      },
    ].filter((item) => !item.hide);
  }
);

const highlight = (content: string) => {
  return getHighlightHTMLByRegExp(
    content,
    props.filter.toLowerCase().split(" "),
    /* !caseSensitive */ false,
    /* className */ "bg-yellow-100"
  );
};

const rowProps = (review: SQLReviewPolicy) => {
  return {
    style: props.customClick ? "cursor: pointer;" : "",
    onClick: () => {
      if (props.customClick) {
        emit("row-click", review);
        return;
      }
    },
  };
};

const hasUpdatePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const hasDeletePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.delete");
});

const toggleReviewEnabled = async (review: SQLReviewPolicy, on: boolean) => {
  await sqlReviewStore.upsertReviewPolicy({
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
