<template>
  <NDataTable
    key="revision-table"
    size="small"
    :columns="columnList"
    :data="revisions"
    :row-key="(revision: Revision) => revision.name"
    :striped="true"
    :row-props="rowProps"
    @update:checked-row-keys="
      (val) => (state.selectedRevisionNameList = new Set(val as string[]))
    "
  />

  <div
    v-if="state.selectedRevisionNameList.size > 0"
    class="sticky w-full flex items-center gap-x-2"
  >
    <NButton size="small" quaternary @click="onDelete">
      <template #icon>
        <Undo2Icon class="w-4 h-auto" />
      </template>
      {{ $t("common.delete") }}
    </NButton>
  </div>
</template>

<script lang="tsx" setup>
import { Undo2Icon } from "lucide-vue-next";
import { type DataTableColumn, NDataTable, NButton, useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import { BBAvatar } from "@/bbkit";
import { useRevisionStore, useUserStore } from "@/store";
import { getDateForPbTimestamp } from "@/types";
import type { Revision } from "@/types/proto/v1/database_service";
import { extractIssueUID, extractUserResourceName } from "@/utils";
import { useDatabaseDetailContext } from "../Database/context";
import HumanizeDate from "../misc/HumanizeDate.vue";

const props = defineProps<{
  revisions: Revision[];
  customClick?: boolean;
  showSelection?: boolean;
}>();

interface LocalState {
  selectedRevisionNameList: Set<string>;
}

const emit = defineEmits<{
  (event: "row-click", name: string): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const revisionStore = useRevisionStore();
const { pagedRevisionTableSessionKey } = useDatabaseDetailContext();
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
      key: "applied-at",
      title: "Applied at",
      width: 128,
      render: (revision) => (
        <HumanizeDate date={getDateForPbTimestamp(revision.createTime)} />
      ),
    },
    {
      key: "version",
      title: t("common.version"),
      width: 128,
      render: (revision) => revision.version,
    },
    {
      key: "filename",
      title: "Filename",
      width: 128,
      render: (revision) => revision.file,
    },
    {
      key: "issue",
      title: t("common.issue"),
      width: "5rem",
      render: (revision) => {
        const uid = extractIssueUID(revision.issue);
        if (!uid) return null;
        return (
          <RouterLink
            to={{
              path: `/${revision.issue}`,
            }}
            custom={true}
          >
            {{
              default: ({ href }: { href: string }) => (
                <a
                  href={href}
                  class="normal-link"
                  onClick={(e: MouseEvent) => e.stopPropagation()}
                >
                  #{uid}
                </a>
              ),
            }}
          </RouterLink>
        );
      },
    },
    {
      key: "statement",
      title: "Statement",
      minWidth: "12rem",
      render: (revision) => {
        return <p class="truncate whitespace-nowrap">{revision.statement}</p>;
      },
    },
    {
      key: "creator",
      title: t("common.creator"),
      width: 128,
      render: (revision) => {
        const creator = creatorOfRevision(revision);
        if (!creator) return null;
        return (
          <div class="flex flex-row items-center overflow-hidden gap-x-1">
            <BBAvatar size="SMALL" username={creator.title} />
            <span class="truncate">{creator.title}</span>
          </div>
        );
      },
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
    },
  };
};

const creatorOfRevision = (revision: Revision) => {
  const email = extractUserResourceName(revision.creator);
  return useUserStore().getUserByEmail(email);
};

const onDelete = () => {
  dialog.warning({
    title: t("database.revision.delete-confirm-dialog.title"),
    content: t("database.revision.delete-confirm-dialog.content"),
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
