<template>
  <NDataTable
    key="changelog-table"
    size="small"
    :columns="columnList"
    :data="changelogs"
    :row-key="(changelog: Changelog) => changelog.name"
    :striped="true"
    :row-props="rowProps"
    :checked-row-keys="selectedChangelogs"
    @update:checked-row-keys="(keys) => $emit('update:selected-changelogs', keys as string[])"
  />
</template>

<script lang="tsx" setup>
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { RouterLink } from "vue-router";
import { BBAvatar } from "@/bbkit";
import { useUserStore } from "@/store";
import { getDateForPbTimestamp } from "@/types";
import type { Changelog } from "@/types/proto/v1/database_service";
import {
  Changelog_Status,
  Changelog_Type,
} from "@/types/proto/v1/database_service";
import {
  extractIssueUID,
  getAffectedTableDisplayName,
  extractUserResourceName,
} from "@/utils";
import {
  changelogLink,
  getAffectedTablesOfChangelog,
  getChangelogChangeType,
} from "@/utils/v1/changelog";
import HumanizeDate from "../misc/HumanizeDate.vue";
import ChangelogStatusIcon from "./ChangelogStatusIcon.vue";

const props = defineProps<{
  changelogs: Changelog[];
  selectedChangelogs?: string[];
  customClick?: boolean;
  showSelection?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:selected-changelogs", value: string[]): void;
  (event: "row-click", changelog: Changelog): void;
}>();

const router = useRouter();
const { t } = useI18n();

const columnList = computed(() => {
  const columns: (DataTableColumn<Changelog> & { hide?: boolean })[] = [
    {
      type: "selection",
      hide: !props.showSelection,
      width: "2rem",
      disabled: (changelog) => {
        return !allowToSelectChangelog(changelog);
      },
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    },
    {
      key: "status",
      width: "2rem",
      render: (changelog) => {
        return (
          <ChangelogStatusIcon class="mx-auto" status={changelog.status} />
        );
      },
    },
    {
      key: "type",
      title: t("changelog.change-type"),
      width: "4rem",
      render: (changelog) => {
        return (
          <div class="flex items-center gap-x-1">
            {getChangelogChangeType(changelog.type)}
          </div>
        );
      },
    },
    {
      key: "issue",
      title: t("common.issue"),
      width: "6rem",
      resizable: true,
      render: (changelog) => {
        const uid = extractIssueUID(changelog.issue);
        if (!uid) return null;
        return (
          <RouterLink
            to={{
              path: `/${changelog.issue}`,
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
      width: "6rem",
      resizable: true,
      render: (changelog) => changelog.version || "-",
    },
    {
      key: "tables",
      title: t("db.tables"),
      width: "15rem",
      resizable: true,
      ellipsis: true,
      render: (changelog) => {
        const tables = getAffectedTablesOfChangelog(changelog);
        if (tables.length === 0) return "-";
        return (
          <p class="space-x-2 truncate">
            {tables.map((table) => (
              <span class={table.dropped ? "text-gray-400 italic" : ""}>
                {getAffectedTableDisplayName(table)}
              </span>
            ))}
          </p>
        );
      },
    },
    {
      key: "SQL",
      title: "SQL",
      resizable: true,
      minWidth: "13rem",
      ellipsis: true,
      render: (changelog) => {
        return <p class="truncate whitespace-nowrap">{changelog.statement}</p>;
      },
    },
    {
      key: "created",
      title: t("common.created-at"),
      width: "7rem",
      resizable: true,
      ellipsis: true,
      render: (changelog) => {
        return (
          <HumanizeDate date={getDateForPbTimestamp(changelog.createTime)} />
        );
      },
    },
    {
      key: "creator",
      title: t("common.creator"),
      width: "8rem",
      resizable: true,
      ellipsis: true,
      render: (changelog) => {
        const creator = useUserStore().getUserByEmail(
          extractUserResourceName(changelog.creator)
        );
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

const rowProps = (changelog: Changelog) => {
  return {
    onClick: (e: MouseEvent) => {
      if (props.customClick) {
        emit("row-click", changelog);
        return;
      }

      const url = changelogLink(changelog);
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};

const allowToSelectChangelog = (changelog: Changelog) => {
  return (
    changelog.status === Changelog_Status.DONE &&
    (changelog.type === Changelog_Type.BASELINE ||
      changelog.type === Changelog_Type.MIGRATE ||
      changelog.type === Changelog_Type.MIGRATE_SDL ||
      changelog.type === Changelog_Type.DATA)
  );
};
</script>
