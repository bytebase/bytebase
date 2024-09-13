import { defineStore } from "pinia";
import type { RouteLocationNormalized } from "vue-router";
import type { RouterSlug } from "@/types";

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
          "/sql-editor/(instances/[0-9a-zA-Z_-]+/databases/[0-9a-zA-Z_-]+)"
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
