import type { RouteRecordRaw } from "vue-router";
import SplashLayout from "@/layouts/SplashLayout.vue";
import { t } from "@/plugins/i18n";

export const SETUP_HOME_MODULE = "setup.home";

const setupRoutes: RouteRecordRaw[] = [
  {
    path: "/setup",
    name: "setup",
    component: SplashLayout,
    children: [
      {
        path: "",
        name: SETUP_HOME_MODULE,
        meta: {
          title: () => t("setup.self"),
          requiredWorkspacePermissionList: () => [
            "bb.settings.get",
            "bb.settings.set",
          ],
        },
        component: () => import("@/views/Setup/Home.vue"),
      },
    ],
  },
];

export default setupRoutes;
