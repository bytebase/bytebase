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
        // /users/:email
        // Total 2 elements, 2nd element is the principal id
        const profileComponents = currentRoute.path.match(
          /\/users\/(\S+@\S+\.\S+)/
        ) || ["/", undefined];
        if (profileComponents[1]) {
          return {
            principalEmail: profileComponents[1],
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
        // /sql-editor/sheet/:sheetSlug
        // match this route first
        const sqlEditorComponents = currentRoute.path.match(
          "/sql-editor/sheet/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];

        if (sqlEditorComponents[1]) {
          return {
            sheetSlug: sqlEditorComponents[1],
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
