import { defineComponent, watch, computed } from "vue";
import { useRouter } from "vue-router";
import { getProjectName } from "@/store/modules/v1/common";
import { PresetRoleType } from "@/types";
import { PROJECT_V1_ROUTE_DETAIL } from "./router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "./router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "./router/sqlEditor";
import { useRecentVisit } from "./router/useRecentVisit";
import { useAppFeature, useProjectV1List } from "./store";
import { hasWorkspaceLevelRole } from "./utils";

export default defineComponent({
  name: "DummyRootView",
  setup() {
    const { lastVisit } = useRecentVisit();

    const router = useRouter();
    const defaultWorkspaceView = useAppFeature(
      "bb.feature.default-workspace-view"
    );

    const hasWorkspaceRole = computed(() => {
      return (
        hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_ADMIN) ||
        hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_DBA)
      );
    });

    const { projectList } = useProjectV1List();

    watch(
      defaultWorkspaceView,
      () => {
        if (defaultWorkspaceView.value === "EDITOR") {
          router.replace({
            name: SQL_EDITOR_HOME_MODULE,
          });
          return;
        }

        const fallback = () => {
          if (hasWorkspaceRole.value || projectList.value.length !== 1) {
            router.replace({
              name: WORKSPACE_ROUTE_LANDING,
            });
          } else {
            router.replace({
              name: PROJECT_V1_ROUTE_DETAIL,
              params: {
                projectId: getProjectName(projectList.value[0].name),
              },
            });
          }
        };

        // Redirect to
        // - /sql-editor if defaultWorkspaceView == 'EDITOR'
        // - lastVisit.path if not empty
        // - /landing as fallback
        if (!lastVisit.value) {
          return fallback();
        }
        const path = lastVisit.value;

        // Ignore all possible root path records
        if (
          path === "" ||
          path === "/" ||
          path.startsWith("?") ||
          path.startsWith("#") ||
          path.startsWith("/?") ||
          path.startsWith("/#") ||
          path.startsWith("/403") ||
          path.startsWith("/404")
        ) {
          return fallback();
        }
        router.replace({
          path,
        });
      },
      {
        immediate: true,
      }
    );
  },
  render() {
    return null;
  },
});
