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
import { CheckIcon, PencilIcon, Trash2Icon, XIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDataTable, NPopconfirm, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import Resource from "@/components/v2/ResourceOccupiedModal/Resource.vue";
import { pushNotification, useSQLReviewStore } from "@/store";
import type { SQLReviewPolicy } from "@/types";
import { getHighlightHTMLByRegExp, hasWorkspacePermissionV2 } from "@/utils";

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
        width: 200,
        resizable: true,
        render: (review: SQLReviewPolicy) => {
          return <div innerHTML={highlight(review.name)}></div>;
        },
      },
      {
        title: t("common.resource"),
        key: "resource",
        resizable: true,
        width: 250,
        render: (review: SQLReviewPolicy) => {
          return (
            <div class="flex flex-wrap gap-2">
              {review.resources.length === 0 && <span>-</span>}
              {review.resources.map((resource) => {
                return (
                  <NTag key={resource} size="small" type="primary">
                    <Resource
                      resource={resource}
                      showPrefix={true}
                      link={true}
                    />
                  </NTag>
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
          return <span>{review.ruleList.length}</span>;
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
        width: "10rem",
        hide: !props.allowEdit,
        render: (review: SQLReviewPolicy) => {
          return (
            <div class="flex items-center gap-x-2">
              {hasDeletePolicyPermission.value && (
                <NPopconfirm
                  positiveButtonProps={{ type: "error" }}
                  onPositiveClick={() => emit("delete", review)}
                >
                  {{
                    trigger: () => (
                      <MiniActionButton type="error">
                        <Trash2Icon />
                      </MiniActionButton>
                    ),
                    default: () => t("common.delete") + ` '${review.name}'?`,
                  }}
                </NPopconfirm>
              )}
              <MiniActionButton onClick={() => emit("edit", review)}>
                <PencilIcon />
              </MiniActionButton>
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
  return hasWorkspacePermissionV2("bb.reviewConfigs.update");
});

const hasDeletePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.reviewConfigs.delete");
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
