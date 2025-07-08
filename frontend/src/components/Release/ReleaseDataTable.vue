<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="sortedReleaseList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(release: ComposedRelease) => release.name"
    :checked-row-keys="Array.from(state.selectedReleaseNameList)"
    :row-props="rowProps"
    @update:checked-row-keys="(val) => (state.selectedReleaseNameList = new Set(val as string[]))"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTag } from "naive-ui";
import { reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import { getTimeForPbTimestampProtoEs, type ComposedRelease } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { getReleaseFileStatement, humanizeTs } from "@/utils";

interface LocalState {
  selectedReleaseNameList: Set<string>;
}

// The max number of files to show in the table cell.
const MAX_SHOW_FILES_COUNT = 3;

const props = withDefaults(
  defineProps<{
    releaseList: ComposedRelease[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
  }>(),
  {
    loading: true,
    bordered: false,
    showSelection: false,
  }
);

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  selectedReleaseNameList: new Set(),
});

const columnList = computed(
  (): (DataTableColumn<ComposedRelease> & { hide?: boolean })[] => {
    const columns: (DataTableColumn<ComposedRelease> & { hide?: boolean })[] = [
      {
        key: "title",
        width: 160,
        title: t("common.title"),
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
        ellipsis: true,
        render: (release) => {
          const showFiles = release.files.slice(0, MAX_SHOW_FILES_COUNT);
          return (
            <div class="flex flex-col items-start gap-1">
              {showFiles.map((file) => (
                <p class="w-full truncate">
                  <NTag class="mr-2" v-if="schemaVersion" size="small" round>
                    {file.version}
                  </NTag>
                  {getReleaseFileStatement(file)}
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
        key: "creator",
        title: t("common.creator"),
        width: 128,
        render: (release) => (
          <div class="flex flex-row items-center overflow-hidden gap-x-2">
            <BBAvatar size="SMALL" username={release.creatorEntity.title} />
            <span class="truncate">{release.creatorEntity.title}</span>
          </div>
        ),
      },
      {
        key: "createTime",
        title: t("common.created-at"),
        width: 128,
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

const rowProps = (release: ComposedRelease) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const url = `/${release.name}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};

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
