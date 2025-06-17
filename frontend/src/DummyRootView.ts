import { defineComponent, watch } from "vue";
import { useRouter } from "vue-router";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { WORKSPACE_ROUTE_LANDING } from "./router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "./router/sqlEditor";
import { useRecentVisit } from "./router/useRecentVisit";
import { useAppFeature } from "./store";

export default defineComponent({
  name: "DummyRootView",
  setup() {
    const { lastVisit } = useRecentVisit();
    const router = useRouter();
    const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

    watch(
      databaseChangeMode,
      () => {
        if (databaseChangeMode.value === DatabaseChangeMode.EDITOR) {
          router.replace({
            name: SQL_EDITOR_HOME_MODULE,
          });
          return;
        }

        const fallback = () => {
          // If no last visit, always redirect to landing page.
          router.replace({
            name: WORKSPACE_ROUTE_LANDING,
          });
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
          path.startsWith("/404") ||
          path.startsWith("/sql-editor")
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
