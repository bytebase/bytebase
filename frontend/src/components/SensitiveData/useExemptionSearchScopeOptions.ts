import type { ComputedRef } from "vue";
import { computed, h } from "vue";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { RichDatabaseName } from "@/components/v2";
import { t } from "@/plugins/i18n";
import { useDatabaseV1Store } from "@/store";
import { getDefaultPagination } from "@/utils";
import { extractDatabaseResourceName } from "@/utils/v1/database";

export function useExemptionSearchScopeOptions(
  projectName: ComputedRef<string>
): ComputedRef<ScopeOption[]> {
  const databaseStore = useDatabaseV1Store();

  return computed((): ScopeOption[] => {
    return [
      {
        id: "database",
        title: t("common.database"),
        search: ({ keyword, nextPageToken: pageToken }) =>
          databaseStore
            .fetchDatabases({
              parent: projectName.value,
              pageToken: pageToken,
              pageSize: getDefaultPagination(),
              filter: { query: keyword },
            })
            .then((resp) => ({
              nextPageToken: resp.nextPageToken,
              options: resp.databases.map<ValueOption>((db) => {
                const { database: dbName } = extractDatabaseResourceName(
                  db.name
                );
                return {
                  value: db.name,
                  keywords: [dbName, db.name],
                  custom: true,
                  render: () =>
                    h(RichDatabaseName, {
                      database: db,
                      showInstance: true,
                      showEngineIcon: true,
                    }),
                };
              }),
            })),
      },
      {
        id: "member",
        title: t("common.members", 1),
      },
      {
        id: "status",
        title: t("common.status"),
        options: [
          {
            value: "active",
            keywords: ["active"],
            render: () => t("common.active"),
          },
          {
            value: "expired",
            keywords: ["expired"],
            render: () => t("sql-editor.expired"),
          },
        ],
      },
    ];
  });
}
