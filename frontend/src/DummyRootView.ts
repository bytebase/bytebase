import { defineComponent, watch } from "vue";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_MY_ISSUES } from "./router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "./router/sqlEditor";
import { useRecentVisit } from "./router/useRecentVisit";
import { useActuatorV1Store, useSettingByName } from "./store";
import { DatabaseChangeMode } from "./types/proto/v1/setting_service";
import { isDev } from "./utils";

export default defineComponent({
  name: "DummyRootView",
  setup() {
    const setting = useSettingByName("bb.workspace.profile");
    const { lastVisit } = useRecentVisit();
    const router = useRouter();
    const actuatorStore = useActuatorV1Store();
    watch(
      setting,
      () => {
        const fallback = () => {
          router.replace({
            name: WORKSPACE_ROUTE_MY_ISSUES,
          });
        };

        actuatorStore.appProfile.mode =
          setting.value?.value?.workspaceProfileSettingValue
            ?.databaseChangeMode === DatabaseChangeMode.EDITOR
            ? "EDITOR"
            : "CONSOLE";

        // Redirect to
        // - /sql-editor if mode == 'EDITOR'
        // - lastVisit.path if not empty
        // - /issues as fallback
        // Only turned on when isDev() is true
        if (!isDev()) {
          return fallback();
        }

        if (actuatorStore.appProfile.mode === "EDITOR") {
          router.replace({
            name: SQL_EDITOR_HOME_MODULE,
          });
          return;
        }

        if (!lastVisit.value) {
          return fallback();
        }
        const { path } = lastVisit.value;
        // Ignore all possible root path records
        if (
          path === "" ||
          path === "/" ||
          path.startsWith("?") ||
          path.startsWith("#") ||
          path.startsWith("/?") ||
          path.startsWith("/#")
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
