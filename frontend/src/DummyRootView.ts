import { defineComponent, watch } from "vue";
import { useRouter } from "vue-router";
import { getProjectName } from "@/store/modules/v1/common";
import { PROJECT_V1_ROUTE_DETAIL } from "./router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "./router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "./router/sqlEditor";
import { useRecentVisit } from "./router/useRecentVisit";
import { useAppFeature, useProjectV1List } from "./store";

export default defineComponent({
  name: "DummyRootView",
  setup() {
    const { lastVisit } = useRecentVisit();

    const router = useRouter();
    const defaultWorkspaceView = useAppFeature(
      "bb.feature.default-workspace-view"
    );
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
          // Redirect to the first project if there are more than one project
          if (projectList.value.length > 1) {
            router.replace({
              name: PROJECT_V1_ROUTE_DETAIL,
              params: {
                projectId: getProjectName(projectList.value[0].name),
              },
            });
          } else {
            // Otherwise, redirect to the landing page.
            router.replace({
              name: WORKSPACE_ROUTE_LANDING,
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
