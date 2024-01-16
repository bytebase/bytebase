import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import ProjectSidebarV1 from "@/components/Project/ProjectSidebarV1.vue";
import InstanceLayout from "@/layouts/InstanceLayout.vue";
import { t } from "@/plugins/i18n";
import { useChangeHistoryStore, useInstanceV1Store } from "@/store";
import { uidFromSlug, idFromSlug } from "@/utils";
import DashboardSidebar from "@/views/DashboardSidebar.vue";

const instanceRoutes: RouteRecordRaw[] = [
  {
    path: "instance",
    name: "workspace.instance",
    meta: {
      title: () => t("common.instances"),
      getQuickActionList: () => {
        return ["quickaction.bb.instance.create"];
      },
    },
    components: {
      content: () => import("@/views/InstanceDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "instance/:instanceSlug",
    components: {
      content: InstanceLayout,
      leftSidebar: DashboardSidebar,
    },
    props: { content: true },
    children: [
      {
        path: "",
        name: "workspace.instance.detail",
        meta: {
          title: (route: RouteLocationNormalized) => {
            const slug = route.params.instanceSlug as string;
            if (slug.toLowerCase() == "new") {
              return t("common.new");
            }
            return useInstanceV1Store().getInstanceByUID(
              String(uidFromSlug(slug))
            ).title;
          },
        },
        component: () => import("@/views/InstanceDetail.vue"),
        props: true,
      },
    ],
  },
  {
    path: "instances/:instance/databases/:database/changeHistories/:changeHistorySlug",
    name: "workspace.database.history.detail",
    meta: {
      title: (route) => {
        const parent = `instances/${route.params.instance}/databases/${route.params.database}`;
        const uid = idFromSlug(route.params.changeHistorySlug as string);
        const name = `${parent}/changeHistories/${uid}`;
        const history = useChangeHistoryStore().getChangeHistoryByName(name);

        return history?.version ?? "";
      },
    },
    components: {
      content: () => import("@/views/ChangeHistoryDetail.vue"),
      leftSidebar: ProjectSidebarV1,
    },
    props: { content: true, leftSidebar: true },
  },
];

export default instanceRoutes;
