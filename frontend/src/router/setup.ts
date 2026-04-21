import type { RouteRecordRaw } from "vue-router";
import SplashLayout from "@/layouts/SplashLayout.vue";
import { t } from "@/plugins/i18n";

export const SETUP_MODULE = "setup";

const setupRoutes: RouteRecordRaw[] = [
  {
    path: "/setup",
    component: SplashLayout,
    children: [
      {
        path: "",
        name: SETUP_MODULE,
        meta: {
          title: () => t("setup.self"),
          requiredPermissionList: () => [
            "bb.settings.get",
            "bb.settings.setWorkspaceProfile",
            "bb.projects.create",
            "bb.roles.list",
            "bb.workspaces.getIamPolicy",
          ],
        },
        component: () => import("@/react/ReactPageMount.vue"),
        props: { page: "SetupPage" },
      },
    ],
  },
];

export default setupRoutes;
