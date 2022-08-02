import { defineStore } from "pinia";
import { RouteLocationNormalized } from "vue-router";
import { RouterSlug } from "@/types";

export const useRouterStore = defineStore("router", {
  // need not to initialize a state since we store everything into localStorage
  // state: () => ({}),

  getters: {
    backPath: () => () => {
      return localStorage.getItem("ui.backPath") || "/";
    },
  },
  actions: {
    setBackPath(backPath: string) {
      localStorage.setItem("ui.backPath", backPath);
      return backPath;
    },
    routeSlug(currentRoute: RouteLocationNormalized): RouterSlug {
      {
        // /u/:principalId
        // Total 2 elements, 2nd element is the principal id
        const profileComponents = currentRoute.path.match(
          "/u/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (profileComponents[1]) {
          return {
            principalId: parseInt(profileComponents[1]),
          };
        }
      }

      {
        // /environment/:environmentSlug
        // Total 2 elements, 2nd element is the issue slug
        const environmentComponents = currentRoute.path.match(
          "/environment/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (environmentComponents[1]) {
          return {
            environmentSlug: environmentComponents[1],
          };
        }
      }

      {
        // /project/:projectSlug/webhook/:hookSlug
        // Total 3 elements, 2nd element is the project slug, 3rd element is the project webhook slug
        const projectComponents = currentRoute.path.match(
          "/project/([0-9a-zA-Z_-]+)/webhook/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined, undefined];
        if (projectComponents[1] && projectComponents[2]) {
          return {
            projectSlug: projectComponents[1],
            projectWebhookSlug:
              projectComponents[2].toLowerCase() == "new"
                ? undefined
                : projectComponents[2],
          };
        }
      }

      {
        // /project/:projectSlug
        // Total 2 elements, 2nd element is the project slug
        const projectComponents = currentRoute.path.match(
          "/project/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (projectComponents[1]) {
          return {
            projectSlug: projectComponents[1],
          };
        }
      }

      {
        // /issue/:issueSlug
        // Total 2 elements, 2nd element is the issue slug
        const issueComponents = currentRoute.path.match(
          "/issue/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (issueComponents[1]) {
          return {
            issueSlug: issueComponents[1],
          };
        }
      }

      {
        // /db/:databaseSlug/table/:tableName
        // Total 3 elements, 2nd element is the database slug, 3rd element is the table name
        const databaseComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)/table/(.+)"
        ) || ["/", undefined, undefined];
        if (databaseComponents[1] && databaseComponents[2]) {
          return {
            databaseSlug: databaseComponents[1],
            tableName: databaseComponents[2],
          };
        }
      }

      {
        // /db/:databaseSlug/data-source/:dataSourceSlug
        // Total 3 elements, 2nd element is the database slug, 3rd element is the data source slug
        const dataSourceComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)/data-source/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined, undefined];
        if (dataSourceComponents[1] && dataSourceComponents[2]) {
          return {
            databaseSlug: dataSourceComponents[1],
            dataSourceSlug: dataSourceComponents[2],
          };
        }
      }

      {
        // /db/:databaseSlug/history/:migrationHistorySlug
        // Total 3 elements, 2nd element is the database slug, 3rd element is the migration history slug
        const migrationHistoryComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)/history/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined, undefined];
        if (migrationHistoryComponents[1] && migrationHistoryComponents[2]) {
          return {
            databaseSlug: migrationHistoryComponents[1],
            migrationHistorySlug: migrationHistoryComponents[2],
          };
        }
      }

      {
        // /db/:databaseSlug
        // Total 2 elements, 2nd element is the database slug
        const databaseComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (databaseComponents[1]) {
          return {
            databaseSlug: databaseComponents[1],
          };
        }
      }

      {
        // /instance/:instanceSlug
        // Total 2 elements, 2nd element is the instance slug
        const instanceComponents = currentRoute.path.match(
          "/instance/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (instanceComponents[1]) {
          return {
            instanceSlug: instanceComponents[1],
          };
        }
      }

      {
        // /setting/version-control/:vcsId
        // Total 2 elements, 2nd element is the version control system id
        const vcsComponents = currentRoute.path.match(
          "/setting/version-control/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (vcsComponents[1]) {
          return {
            vcsSlug: vcsComponents[1],
          };
        }
      }

      {
        // /setting/sql-review/:sqlReviewPolicyId
        // Total 2 elements, 2nd element is the SQL review id
        const sqlReviewComponents = currentRoute.path.match(
          "/setting/sql-review/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (sqlReviewComponents[1]) {
          return {
            sqlReviewPolicySlug: sqlReviewComponents[1],
          };
        }
      }

      {
        // /sql-editor/:connectionSlug/:sheetSlug
        // match this route first
        const sqlEditorComponents = currentRoute.path.match(
          "/sql-editor/([0-9a-zA-Z_-]+)/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined, undefined];

        if (sqlEditorComponents[1] && sqlEditorComponents[2]) {
          return {
            connectionSlug: sqlEditorComponents[1],
            sheetSlug: sqlEditorComponents[2],
          };
        }
      }

      {
        // /sql-editor/:connectionSlug
        const sqlEditorComponents = currentRoute.path.match(
          "/sql-editor/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];

        if (sqlEditorComponents[1]) {
          return {
            connectionSlug: sqlEditorComponents[1],
          };
        }
      }

      return {};
    },
  },
});
