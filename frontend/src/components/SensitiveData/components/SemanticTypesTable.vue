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
import { NPopconfirm, NSelect, NInput, NDataTable } from "naive-ui";
import type { SelectOption, DataTableColumn } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import { useSettingV1Store } from "@/store";
import {
  Algorithm,
  type SemanticTypeSetting_SemanticType,
} from "@/types/proto/v1/setting_service";
import { isDev } from "@/utils";

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
const settingStore = useSettingV1Store();

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
    (data) => (data.item.algorithms = maskingAlgorithm)
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
  if (isDev()) {
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
              {item.item.algorithms?.title ??
                t("settings.sensitive-data.algorithms.default")}
            </h3>
            {!props.readonly && (
              <MiniActionButton
                onClick={() => {
                  state.pendingEditAlgorithm =
                    item.item.algorithms ??
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
  } else {
    columns.push(
      {
        key: "full-masking-algorithm",
        title: t(
          "settings.sensitive-data.semantic-types.table.full-masking-algorithm"
        ),
        width: "minmax(min-content, auto)",
        render: (item, row) => {
          if (item.mode === "NORMAL") {
            return (
              <h3>
                {getAlgorithmById(item.item.fullMaskAlgorithmId)?.label ??
                  t("settings.sensitive-data.algorithms.default")}
              </h3>
            );
          }

          return (
            <NSelect
              value={item.item.fullMaskAlgorithmId}
              options={algorithmList.value}
              consistentMenuWidth={false}
              placeholder={t("settings.sensitive-data.algorithms.default")}
              fallbackOption={(_: string) => ({
                label: t("settings.sensitive-data.algorithms.default"),
                value: "",
              })}
              clearable
              size="small"
              style="min-width: 7rem; width: auto; overflow-x: hidden"
              onUpdateValue={(val) =>
                onInput(row, (data) => (data.item.fullMaskAlgorithmId = val))
              }
            />
          );
        },
      },
      {
        key: "partial-masking-algorithm",
        title: t(
          "settings.sensitive-data.semantic-types.table.partial-masking-algorithm"
        ),
        width: "minmax(min-content, auto)",
        render: (item, row) => {
          if (item.mode === "NORMAL") {
            return (
              <h3>
                {getAlgorithmById(item.item.partialMaskAlgorithmId)?.label ??
                  t("settings.sensitive-data.algorithms.default")}
              </h3>
            );
          }

          return (
            <NSelect
              value={item.item.partialMaskAlgorithmId}
              options={algorithmList.value}
              consistentMenuWidth={false}
              placeholder={t("settings.sensitive-data.algorithms.default")}
              fallbackOption={(_: string) => ({
                label: t("settings.sensitive-data.algorithms.default"),
                value: "",
              })}
              clearable
              size="small"
              style="min-width: 7rem; width: auto; overflow-x: hidden"
              onUpdateValue={(val) =>
                onInput(row, (data) => (data.item.partialMaskAlgorithmId = val))
              }
            />
          );
        },
      }
    );
  }
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
    onClick: (e: MouseEvent) => {
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

const algorithmList = computed((): SelectOption[] => {
  const list = (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  ).map((algorithm) => ({
    label: algorithm.title,
    value: algorithm.id,
  }));

  list.unshift({
    label: t("settings.sensitive-data.algorithms.default"),
    value: "",
  });

  return list;
});

const getAlgorithmById = (algorithmId: string) => {
  return algorithmList.value.find((a) => a.value === algorithmId);
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
