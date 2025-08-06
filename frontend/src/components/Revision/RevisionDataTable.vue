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
    <NButton size="small" quaternary @click="onDelete">
      <template #icon>
        <TrashIcon class="w-4 h-auto" />
      </template>
      {{ $t("common.delete") }}
    </NButton>
  </div>
</template>

<script lang="tsx" setup>
import { TrashIcon } from "lucide-vue-next";
import { type DataTableColumn, NDataTable, NButton, useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useRevisionStore } from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import {
  extractIssueUID,
  extractProjectResourceName,
  issueV1Slug,
} from "@/utils";
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
      key: "issue",
      title: t("common.issue"),
      width: 96,
      render: (revision) => {
        const uid = extractIssueUID(revision.issue);
        if (!uid) return "-";
        return (
          <RouterLink
            to={{
              name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
              params: {
                projectId: extractProjectResourceName(revision.issue),
                issueSlug: issueV1Slug(revision.issue),
              },
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
      key: "version",
      title: t("common.version"),
      width: 128,
      render: (revision) => revision.version,
    },
    {
      key: "statement",
      title: t("common.statement"),
      resizable: true,
      minWidth: "13rem",
      ellipsis: true,
      render: (revision) => {
        return <p class="truncate whitespace-nowrap">{revision.statement}</p>;
      },
    },
    {
      key: "created-at",
      title: t("common.created-at"),
      width: 128,
      resizable: true,
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
