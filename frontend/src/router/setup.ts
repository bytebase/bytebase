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
          requiredPermissionList: () => ["bb.settings.get", "bb.settings.set"],
        },
        component: () => import("@/views/Setup/Setup.vue"),
      },
    ],
  },
];

export default setupRoutes;
