<template>
  <NDataTable
    :size="size"
    :data="semanticItemList"
    :columns="columnList"
    :striped="true"
    :bordered="true"
    :row-props="rowProps"
    :row-key="(item: SemanticItem) => item.item.id"
  />
</template>

<script lang="tsx" setup>
import { InfoIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  Algorithm,
  SemanticTypeSetting_SemanticType,
} from "@/types/proto-es/v1/setting_service_pb";

export interface SemanticItem {
  mode: "NORMAL";
  dirty: boolean;
  item: SemanticTypeSetting_SemanticType;
}

const props = withDefaults(
  defineProps<{
    size?: "small" | "medium" | "large";
    semanticItemList: SemanticItem[];
    rowClickable: boolean;
  }>(),
  { size: "medium" }
);

const emit = defineEmits<{
  (event: "select", id: string): void;
}>();

const { t } = useI18n();

type MaskingType = "full-mask" | "range-mask" | "md5-mask" | "inner-outer-mask";

const getMaskingType = (
  algorithm: Algorithm | undefined
): MaskingType | undefined => {
  if (!algorithm?.mask) return undefined;
  switch (algorithm.mask.case) {
    case "fullMask":
      return "full-mask";
    case "rangeMask":
      return "range-mask";
    case "innerOuterMask":
      return "inner-outer-mask";
    case "md5Mask":
      return "md5-mask";
    default:
      return undefined;
  }
};

const isBuiltinSemanticType = (item: SemanticTypeSetting_SemanticType) => {
  return item.id.startsWith("bb.");
};

const columnList = computed(() => {
  const columns: DataTableColumn<SemanticItem>[] = [
    {
      key: "icon",
      title: t("settings.sensitive-data.semantic-types.table.icon"),
      width: 80,
      align: "center",
      render: (item) => {
        if (item.item.icon) {
          return (
            <div class="flex items-center justify-center">
              <img src={item.item.icon} class="w-6 h-6 object-contain" alt="" />
            </div>
          );
        }
        return (
          <div class="flex items-center justify-center">
            <span class="text-gray-400">-</span>
          </div>
        );
      },
    },
    {
      key: "id",
      title: "ID",
      width: 150,
      resizable: true,
      ellipsis: { tooltip: true },
      render: (item) => item.item.id,
    },
    {
      key: "title",
      title: t("settings.sensitive-data.semantic-types.table.semantic-type"),
      width: "minmax(min-content, auto)",
      resizable: true,
      ellipsis: { tooltip: true },
      render: (item) => item.item.title,
    },
    {
      key: "description",
      title: t("settings.sensitive-data.semantic-types.table.description"),
      width: 200,
      resizable: true,
      ellipsis: { tooltip: true },
      render: (item) => item.item.description,
    },
    {
      key: "algorithm",
      title: t(
        "settings.sensitive-data.semantic-types.table.masking-algorithm"
      ),
      width: "minmax(min-content, auto)",
      resizable: true,
      render: (item) => {
        return (
          <div class="flex items-center gap-x-1">
            {isBuiltinSemanticType(item.item) ? (
              <h3>
                {t(
                  `dynamic.settings.sensitive-data.semantic-types.template.${item.item.id.split(".").join("-")}.title`
                )}
              </h3>
            ) : (
              <h3>
                {getMaskingType(item.item.algorithm)
                  ? t(
                      `settings.sensitive-data.algorithms.${getMaskingType(item.item.algorithm)?.toLowerCase()}.self`
                    )
                  : "N/A"}
              </h3>
            )}
            {isBuiltinSemanticType(item.item) && (
              <NTooltip>
                {{
                  trigger: () => <InfoIcon class="w-4" />,
                  default: () => (
                    <div class="whitespace-pre-line">
                      {t(
                        `dynamic.settings.sensitive-data.semantic-types.template.${item.item.id.split(".").join("-")}.algorithm.description`
                      )}
                    </div>
                  ),
                }}
              </NTooltip>
            )}
          </div>
        );
      },
    },
  ];

  return columns;
});

const rowProps = (item: SemanticItem) => {
  return {
    style: props.rowClickable ? "cursor: pointer;" : "",
    onClick: () => {
      if (!props.rowClickable) {
        return;
      }
      emit("select", item.item.id);
    },
  };
};
</script>
