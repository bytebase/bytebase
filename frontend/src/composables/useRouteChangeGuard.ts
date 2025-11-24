import { useEventListener } from "@vueuse/core";
import { type Ref } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { t } from "@/plugins/i18n";

export const useRouteChangeGuard = (
  isEditing: Ref<boolean>,
  content?: string
) => {
  useEventListener("beforeunload", (e) => {
    if (!isEditing.value) {
      return;
    }
    e.returnValue = content ?? t("common.leave-without-saving");
    return e.returnValue;
  });

  onBeforeRouteLeave((to, from, next) => {
    if (isEditing.value) {
      if (!window.confirm(content ?? t("common.leave-without-saving"))) {
        return;
      }
    }
    next();
  });
};
