<template>
  <NDataTable
    :data="semanticItemList"
    :columns="columnList"
    :striped="true"
    :bordered="true"
    :row-props="rowProps"
    :row-key="(item: SemanticItem) => item.item.id"
  />

  <MaskingAlgorithmsCreateDrawer
    :show="state.showAlgorithmDrawer"
    :algorithm="state.pendingEditAlgorithm"
    :readonly="readonly"
    @apply="onAlgorithmUpsert"
    @dismiss="onDrawerDismiss"
  />
</template>

<script lang="tsx" setup>
import { CheckIcon, PencilIcon, TrashIcon, Undo2Icon } from "lucide-vue-next";
import { NPopconfirm, NInput, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import {
  Algorithm,
  type SemanticTypeSetting_SemanticType,
} from "@/types/proto/v1/setting_service";
import MaskingAlgorithmsCreateDrawer from "./MaskingAlgorithmsCreateDrawer.vue";

type SemanticItemMode = "NORMAL" | "CREATE" | "EDIT";

export interface SemanticItem {
  mode: SemanticItemMode;
  dirty: boolean;
  item: SemanticTypeSetting_SemanticType;
}

interface LocalState {
  showAlgorithmDrawer: boolean;
  pendingEditAlgorithm: Algorithm;
  pendingEditSemanticIndex?: number;
  processing: boolean;
}

const props = defineProps<{
  readonly: boolean;
  semanticItemList: SemanticItem[];
  rowClickable: boolean;
}>();

const emit = defineEmits<{
  (event: "select", id: string): void;
  (event: "remove", index: number): void;
  (event: "cancel", index: number): void;
  (event: "confirm", index: number): void;
}>();

const state = reactive<LocalState>({
  showAlgorithmDrawer: false,
  pendingEditAlgorithm: Algorithm.fromPartial({
    id: uuidv4(),
  }),
  processing: false,
});

const { t } = useI18n();

const onDrawerDismiss = () => {
  state.pendingEditSemanticIndex = undefined;
  state.showAlgorithmDrawer = false;
};

const onAlgorithmUpsert = async (maskingAlgorithm: Algorithm) => {
  if (state.pendingEditSemanticIndex === undefined) {
    return;
  }
  onInput(
    state.pendingEditSemanticIndex,
    (data) => (data.item.algorithm = maskingAlgorithm)
  );
  emit("confirm", state.pendingEditSemanticIndex);
  onDrawerDismiss();
};

const columnList = computed(() => {
  const columns: DataTableColumn<SemanticItem>[] = [
    {
      key: "title",
      title: t("settings.sensitive-data.semantic-types.table.semantic-type"),
      render: (item, row) => {
        if (item.mode === "NORMAL") {
          return <h3 class="break-normal">{item.item.title}</h3>;
        }
        return (
          <NInput
            value={item.item.title}
            size="small"
            type="text"
            placeholder={t(
              "settings.sensitive-data.semantic-types.table.semantic-type"
            )}
            onInput={(val: string) =>
              onInput(row, (data) => (data.item.title = val))
            }
          />
        );
      },
    },
    {
      key: "description",
      title: t("settings.sensitive-data.semantic-types.table.description"),
      width: "minmax(min-content, auto)",
      render: (item, row) => {
        if (item.mode === "NORMAL") {
          return <h3>{item.item.description}</h3>;
        }
        return (
          <NInput
            value={item.item.description}
            size="small"
            type="text"
            placeholder={t(
              "settings.sensitive-data.semantic-types.table.description"
            )}
            onInput={(val: string) =>
              onInput(row, (data) => (data.item.description = val))
            }
          />
        );
      },
    },
  ];
  
  columns.push({
    key: "algorithm",
    title: t(
      "settings.sensitive-data.semantic-types.table.masking-algorithm"
    ),
    width: "minmax(min-content, auto)",
    render: (item, row) => {
      return (
        <div class="flex items-center space-x-1">
          <h3>
            {item.item.algorithm?.title ??
              t("settings.sensitive-data.algorithms.default")}
          </h3>
          {!props.readonly && (
            <MiniActionButton
              onClick={() => {
                state.pendingEditAlgorithm =
                  item.item.algorithm ??
                  Algorithm.fromPartial({
                    id: uuidv4(),
                  });
                state.pendingEditSemanticIndex = row;
                state.showAlgorithmDrawer = true;
              }}
            >
              <PencilIcon class="w-4 h-4" />
            </MiniActionButton>
          )}
        </div>
      );
    },
  });
  
  if (!props.readonly) {
    // operation.
    columns.push({
      key: "operation",
      title: "",
      width: "minmax(min-content, auto)",
      render: (item, row) => {
        return (
          <div>
            {item.mode === "EDIT" && (
              <NPopconfirm onPositiveClick={() => emit("remove", row)}>
                {{
                  trigger: () => {
                    return (
                      <MiniActionButton>
                        <TrashIcon class="w-4 h-4" />
                      </MiniActionButton>
                    );
                  },
                  default: () => (
                    <div class="whitespace-nowrap">
                      {t("settings.sensitive-data.semantic-types.table.delete")}
                    </div>
                  ),
                }}
              </NPopconfirm>
            )}
            {item.mode !== "NORMAL" && (
              <MiniActionButton onClick={() => emit("cancel", row)}>
                <Undo2Icon class="w-4 h-4" />
              </MiniActionButton>
            )}
            {item.mode !== "NORMAL" && (
              <MiniActionButton
                type={"primary"}
                disabled={isConfirmDisabled(item)}
                onClick={() => emit("confirm", row)}
              >
                <CheckIcon class="w-4 h-4" />
              </MiniActionButton>
            )}
            {item.mode === "NORMAL" && (
              <MiniActionButton onClick={() => (item.mode = "EDIT")}>
                <PencilIcon class="w-4 h-4" />
              </MiniActionButton>
            )}
          </div>
        );
      },
    });
  }
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

const onInput = (index: number, callback: (item: SemanticItem) => void) => {
  const item = props.semanticItemList[index];
  if (!item) {
    return;
  }
  callback(item);
  item.dirty = true;
};

const isConfirmDisabled = (data: SemanticItem): boolean => {
  if (!data.item.title) {
    return true;
  }
  if (data.mode === "EDIT" && !data.dirty) {
    return true;
  }
  return false;
};
</script>
