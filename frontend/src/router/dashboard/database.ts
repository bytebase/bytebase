import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import ProjectSidebarV1 from "@/components/Project/ProjectSidebarV1.vue";
import DatabaseLayout from "@/layouts/DatabaseLayout.vue";
import { t } from "@/plugins/i18n";
import { useDatabaseV1Store } from "@/store";
import { uidFromSlug } from "@/utils";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import { DATABASE_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const DATABASE_ROUTE_DETAIL = `${DATABASE_ROUTE_DASHBOARD}.detail`;

const databaseRoutes: RouteRecordRaw[] = [
  {
    path: "db",
    name: DATABASE_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.databases"),
      getQuickActionList: () => {
        return ["quickaction.bb.database.create"];
      },
    },
    components: {
      content: () => import("@/views/DatabaseDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "db/:databaseSlug",
    components: {
      content: DatabaseLayout,
      leftSidebar: ProjectSidebarV1,
    },
    props: { content: true, leftSidebar: true },
    children: [
      {
        path: "",
        name: DATABASE_ROUTE_DETAIL,
        meta: {
          title: (route: RouteLocationNormalized) => {
            const slug = route.params.databaseSlug as string;
            if (slug.toLowerCase() == "new") {
              return t("common.new");
            }
            return useDatabaseV1Store().getDatabaseByUID(
              String(uidFromSlug(slug))
            ).databaseName;
          },
        },
        component: () => import("@/views/DatabaseDetail.vue"),
        props: true,
      },
    ],
  },
];

export default databaseRoutes;
