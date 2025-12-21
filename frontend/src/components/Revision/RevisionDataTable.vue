<template>
  <NDataTable
    key="revision-table"
    size="small"
    :columns="columnList"
    :data="revisions"
    :row-key="(revision: Revision) => revision.name"
    :striped="true"
    :row-props="rowProps"
    :loading="loading"
    @update:checked-row-keys="
      (val) => (state.selectedRevisionNameList = new Set(val as string[]))
    "
  />

  <div
    v-if="state.selectedRevisionNameList.size > 0"
    class="sticky w-full flex items-center gap-x-2"
  >
    <NButton size="small" type="error" quaternary @click="onDelete">
      <template #icon>
        <TrashIcon class="w-4 h-auto" />
      </template>
      {{ $t("common.delete") }}
    </NButton>
  </div>
</template>

<script lang="tsx" setup>
import { TrashIcon } from "lucide-vue-next";
import { type DataTableColumn, NButton, NDataTable, useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useRevisionStore } from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  type Revision,
  Revision_Type,
} from "@/types/proto-es/v1/revision_service_pb";
import { useDatabaseDetailContext } from "../Database/context";
import HumanizeDate from "../misc/HumanizeDate.vue";

const props = defineProps<{
  revisions: Revision[];
  customClick?: boolean;
  showSelection?: boolean;
  loading?: boolean;
}>();

interface LocalState {
  selectedRevisionNameList: Set<string>;
}

const emit = defineEmits<{
  (event: "row-click", name: string): void;
}>();

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const revisionStore = useRevisionStore();
const { pagedRevisionTableSessionKey, database } = useDatabaseDetailContext();
const state = reactive<LocalState>({
  selectedRevisionNameList: new Set(),
});

const columnList = computed(() => {
  const columns: (DataTableColumn<Revision> & { hide?: boolean })[] = [
    {
      type: "selection",
      width: 40,
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
      hide: !props.showSelection,
    },
    {
      key: "version",
      title: t("common.version"),
      width: 160,
      resizable: true,
      render: (revision) => revision.version,
    },
    {
      key: "type",
      title: t("common.type"),
      width: 128,
      resizable: true,
      render: (revision) => Revision_Type[revision.type],
    },
    {
      key: "created-at",
      title: t("common.created-at"),
      width: 128,
      render: (revision) => (
        <HumanizeDate
          date={getDateForPbTimestampProtoEs(revision.createTime)}
        />
      ),
    },
  ];
  return columns.filter((col) => !col.hide);
});

const rowProps = (revision: Revision) => {
  return {
    onClick: (e: MouseEvent) => {
      if (props.customClick) {
        emit("row-click", revision.name);
        return;
      }

      const url = `/${database.value.project}/${revision.name}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};

const onDelete = () => {
  dialog.warning({
    title: t("bbkit.confirm-button.sure-to-delete"),
    content: t("database.revision.delete-confirm-dialog"),
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onPositiveClick: async () => {
      for (const name of state.selectedRevisionNameList) {
        await revisionStore.deleteRevision(name);
      }
      pagedRevisionTableSessionKey.value = `bb.paged-revision-table.${Date.now()}`;
    },
  });
};
</script>
