import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  FormField,
  FormFieldGroup,
  FormSection,
} from "@/react/components/ui/form";
import { useAppStore } from "@/react/stores/app";
import type { SectionHandle } from "./useSettingSection";

interface ProductImprovementSectionProps {
  allowEdit: boolean;
  onDirtyChange: () => void;
}

interface State {
  enableMetricCollection: boolean;
}

export const ProductImprovementSection = forwardRef<
  SectionHandle,
  ProductImprovementSectionProps
>(function ProductImprovementSection({ allowEdit, onDirtyChange }, ref) {
  const { t } = useTranslation();

  const getInitialState = useCallback(
    (): State => ({
      enableMetricCollection: useAppStore.getState().getWorkspaceProfile()
        .enableMetricCollection,
    }),
    []
  );

  const [state, setState] = useState<State>(getInitialState);

  const isDirty = useCallback(
    () =>
      state.enableMetricCollection !== getInitialState().enableMetricCollection,
    [state, getInitialState]
  );

  const revert = useCallback(() => {
    setState(getInitialState());
  }, [getInitialState]);

  const update = useCallback(async () => {
    await useAppStore.getState().updateWorkspaceProfile({
      payload: {
        enableMetricCollection: state.enableMetricCollection,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.enable_metric_collection"],
      }),
    });
  }, [state]);

  const title = t("settings.general.workspace.product-improvement.self");

  useImperativeHandle(ref, () => ({ isDirty, revert, update }));

  useEffect(() => {
    onDirtyChange();
  }, [state, onDirtyChange]);

  return (
    <FormSection id="product-improvement" title={title}>
      <FormFieldGroup>
        <FormField
          title={
            <span className="flex items-start gap-x-3">
              <Checkbox
                checked={state.enableMetricCollection}
                className="mt-1"
                disabled={!allowEdit}
                onCheckedChange={(checked) =>
                  setState((s) => ({
                    ...s,
                    enableMetricCollection: checked,
                  }))
                }
              />
              {t("settings.general.workspace.product-improvement.participate")}
            </span>
          }
          description={t(
            "settings.general.workspace.product-improvement.description"
          )}
        />
      </FormFieldGroup>
    </FormSection>
  );
});
