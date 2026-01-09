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
    :checked-row-keys="[...selectedRevisionNames]"
    @update:checked-row-keys="
      (keys) => (selectedRevisionNames = new Set(keys as string[]))
    "
  />

  <div
    v-if="showSelection && selectedRevisionNames.size > 0"
    class="sticky bottom-0 w-full flex items-center gap-x-2 bg-white py-2"
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
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useRevisionStore } from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import { getRevisionType, revisionLink } from "@/utils/v1/revision";
import HumanizeDate from "../misc/HumanizeDate.vue";

const props = defineProps<{
  revisions: Revision[];
  customClick?: boolean;
  showSelection?: boolean;
  loading?: boolean;
}>();

const emit = defineEmits<{
  (event: "row-click", revision: Revision): void;
  (event: "delete"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const revisionStore = useRevisionStore();

const selectedRevisionNames = ref<Set<string>>(new Set());

const columnList = computed(() => {
  const columns: (DataTableColumn<Revision> & { hide?: boolean })[] = [
    {
      type: "selection",
      hide: !props.showSelection,
      width: "2rem",
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    },
    {
      key: "version",
      title: t("common.version"),
      minWidth: 160,
      resizable: true,
      render: (revision) => revision.version,
    },
    {
      key: "type",
      title: t("common.type"),
      width: 128,
      render: (revision) => getRevisionType(revision.type),
    },
    {
      key: "created-at",
      title: t("common.created-at"),
      width: 180,
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
        emit("row-click", revision);
        return;
      }

      const url = revisionLink(revision);
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
      for (const name of selectedRevisionNames.value) {
        await revisionStore.deleteRevision(name);
      }
      selectedRevisionNames.value = new Set();
      emit("delete");
    },
  });
};
</script>
