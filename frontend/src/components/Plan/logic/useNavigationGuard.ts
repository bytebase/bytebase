import { useDialog } from "naive-ui";
import { useI18n } from "vue-i18n";
import { useEditorState } from "./useEditorState";

export const useNavigationGuard = () => {
  const dialog = useDialog();
  const { t } = useI18n();
  const editorState = useEditorState();

  const confirmNavigation = (): Promise<boolean> => {
    return new Promise((resolve) => {
      // If not editing, allow navigation
      if (!editorState.isEditing.value) {
        resolve(true);
        return;
      }

      // Show confirmation dialog when in editing mode
      dialog.warning({
        title: t("plan.editor.unsaved-changes"),
        content: t("plan.editor.unsaved-changes-message"),
        positiveText: t("common.leave"),
        negativeText: t("common.stay"),
        onPositiveClick: () => {
          // Stop editing when user chooses to leave
          editorState.setEditingState(false);
          resolve(true);
        },
        onNegativeClick: () => resolve(false),
        onClose: () => resolve(false),
      });
    });
  };

  return { confirmNavigation };
};
