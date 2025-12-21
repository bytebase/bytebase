<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="sortedReleaseList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(release) => release.name"
    :checked-row-keys="Array.from(state.selectedReleaseNameList)"
    :row-props="rowProps"
    @update:checked-row-keys="handleSelectionChange"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTag } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import { humanizeTs } from "@/utils";

interface LocalState {
  selectedReleaseNameList: Set<string>;
}

// The max number of files to show in the table cell.
const MAX_SHOW_FILES_COUNT = 3;

const props = withDefaults(
  defineProps<{
    releaseList: Release[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
    singleSelect?: boolean;
    selectedRowKeys?: string[];
  }>(),
  {
    loading: true,
    bordered: false,
    showSelection: false,
    singleSelect: false,
    selectedRowKeys: () => [],
  }
);

const emit = defineEmits<{
  (event: "update:checked-row-keys", keys: string[]): void;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  selectedReleaseNameList: new Set(props.selectedRowKeys),
});

const columnList = computed(
  (): (DataTableColumn<Release> & { hide?: boolean })[] => {
    const columns: (DataTableColumn<Release> & { hide?: boolean })[] = [
      {
        type: "selection",
        hide: !props.showSelection,
        multiple: !props.singleSelect,
      },
      {
        key: "title",
        width: 300,
        title: t("common.title"),
        resizable: true,
        render: (release) => {
          return (
            <p class="inline-flex w-full">
              {release.title ? (
                <span class="shrink truncate">{release.title}</span>
              ) : (
                <span class="shrink truncate italic opacity-60">
                  {t("common.untitled")}
                </span>
              )}
              {release.state === State.DELETED && (
                <NTag class="shrink-0" type="warning" size="small" round>
                  {t("common.archived")}
                </NTag>
              )}
            </p>
          );
        },
      },
      {
        key: "files",
        title: t("release.files"),
        resizable: true,
        ellipsis: true,
        render: (release) => {
          const showFiles = release.files.slice(0, MAX_SHOW_FILES_COUNT);
          return (
            <div class="flex flex-col items-start gap-1">
              {showFiles.map((file) => (
                <p class="w-full truncate">
                  {file.version && (
                    <NTag class="mr-2" size="small" round>
                      {file.version}
                    </NTag>
                  )}
                  {file.path}
                </p>
              ))}
              {release.files.length > MAX_SHOW_FILES_COUNT && (
                <p class="text-gray-400 text-xs italic">
                  {t("release.total-files", { count: release.files.length })}
                </p>
              )}
            </div>
          );
        },
      },
      {
        key: "createTime",
        title: t("common.created-at"),
        width: 128,
        resizable: true,
        render: (release) =>
          humanizeTs(
            getTimeForPbTimestampProtoEs(release.createTime, 0) / 1000
          ),
      },
    ];
    return columns.filter((column) => !column.hide);
  }
);

const sortedReleaseList = computed(() => {
  return props.releaseList;
});

const rowProps = (release: Release) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      // Check if we're in selection mode
      if (props.showSelection) {
        // Don't toggle if clicking on the checkbox itself
        const target = e.target as HTMLElement;
        if (target.closest(".n-checkbox")) {
          return;
        }

        const releaseName = release.name;

        if (props.singleSelect) {
          // For single select, simply select the clicked row
          state.selectedReleaseNameList = new Set([releaseName]);
          emit("update:checked-row-keys", [releaseName]);
        } else {
          // For multi-select, toggle the selection
          if (state.selectedReleaseNameList.has(releaseName)) {
            state.selectedReleaseNameList.delete(releaseName);
          } else {
            state.selectedReleaseNameList.add(releaseName);
          }
          emit(
            "update:checked-row-keys",
            Array.from(state.selectedReleaseNameList)
          );
        }
      } else {
        // Navigation mode when not in selection mode
        const url = `/${release.name}`;
        if (e.ctrlKey || e.metaKey) {
          window.open(url, "_blank");
        } else {
          router.push(url);
        }
      }
    },
  };
};

const handleSelectionChange = (val: Array<string | number>) => {
  const stringKeys = val.map((v) => String(v));
  if (props.singleSelect && stringKeys.length > 1) {
    // For single select, keep only the most recently selected item
    const newSelection = stringKeys[stringKeys.length - 1];
    state.selectedReleaseNameList = new Set([newSelection]);
    emit("update:checked-row-keys", [newSelection]);
  } else {
    state.selectedReleaseNameList = new Set(stringKeys);
    emit("update:checked-row-keys", stringKeys);
  }
};

// Watch for external selection changes
watch(
  () => props.selectedRowKeys,
  (newKeys) => {
    state.selectedReleaseNameList = new Set(newKeys);
  }
);

// Clean up selected items that are no longer in the list
watch(
  () => props.releaseList,
  (list) => {
    const oldReleaseNames = Array.from(state.selectedReleaseNameList.values());
    const newReleaseNames = new Set(list.map((release) => release.name));
    oldReleaseNames.forEach((name) => {
      if (!newReleaseNames.has(name)) {
        state.selectedReleaseNameList.delete(name);
      }
    });
  }
);
</script>
