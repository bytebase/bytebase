import type { RouteRecordRaw } from "vue-router";
import SplashLayout from "@/layouts/SplashLayout.vue";
import { t } from "@/plugins/i18n";

export const SETUP_WORKSPACE_MODE_MODULE = "setup.workspace-mode";

const setupRoutes: RouteRecordRaw[] = [
  {
    path: "/setup",
    name: "setup",
    component: SplashLayout,
    children: [
      {
        path: "mode",
        name: SETUP_WORKSPACE_MODE_MODULE,
        meta: {
          title: () => `${t("setup.self")} | ${"setup.workspace-mode"}`,
          requiredPermissionList: () => ["bb.settings.get", "bb.settings.set"],
        },
        component: () => import("@/views/Setup/WorkspaceMode.vue"),
      },
    ],
  },
];

export default setupRoutes;
