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
import { useSettingV1Store } from "@/store";
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
  const settingV1Store = useSettingV1Store();

  const getInitialState = useCallback(
    (): State => ({
      enableMetricCollection:
        settingV1Store.workspaceProfile.enableMetricCollection,
    }),
    [settingV1Store]
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
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        enableMetricCollection: state.enableMetricCollection,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.enable_metric_collection"],
      }),
    });
  }, [state, settingV1Store]);

  const title = t("settings.general.workspace.product-improvement.self");

  useImperativeHandle(ref, () => ({ isDirty, revert, update }));

  useEffect(() => {
    onDirtyChange();
  }, [state, onDirtyChange]);

  return (
    <div id="product-improvement" className="py-6 lg:flex">
      <div className="text-left lg:w-1/4">
        <h1 className="text-2xl font-bold">{title}</h1>
      </div>
      <div className="flex-1 lg:px-4">
        <label className="flex items-start gap-x-3 cursor-pointer">
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
          <div className="flex flex-col gap-1">
            <div className="text-base font-semibold">
              {t("settings.general.workspace.product-improvement.participate")}
            </div>
            <div className="textinfolabel">
              {t("settings.general.workspace.product-improvement.description")}
            </div>
          </div>
        </label>
      </div>
    </div>
  );
});
