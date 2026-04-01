import { create } from "@bufbuild/protobuf";
import { Check, Info, Pencil, Plus, Trash2, Undo2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { getSemanticTemplateList } from "@/types";
import type {
  Algorithm,
  SemanticTypeSetting_SemanticType,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  Algorithm_InnerOuterMask_MaskType,
  Algorithm_InnerOuterMaskSchema,
  Algorithm_RangeMask_SliceSchema,
  AlgorithmSchema,
  Algorithm_FullMaskSchema as FullMaskSchema,
  Algorithm_MD5MaskSchema as MD5MaskSchema,
  Algorithm_RangeMaskSchema as RangeMaskSchema,
  SemanticTypeSetting_SemanticTypeSchema,
  Setting_SettingName,
  SettingValueSchema as SettingSettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

type SemanticItemMode = "NORMAL" | "CREATE" | "EDIT";

interface SemanticItem {
  mode: SemanticItemMode;
  dirty: boolean;
  item: SemanticTypeSetting_SemanticType;
}

type MaskingType = "full-mask" | "range-mask" | "md5-mask" | "inner-outer-mask";

function getMaskingType(
  algorithm: Algorithm | undefined
): MaskingType | undefined {
  if (!algorithm?.mask) return undefined;
  switch (algorithm.mask.case) {
    case "fullMask":
      return "full-mask";
    case "rangeMask":
      return "range-mask";
    case "innerOuterMask":
      return "inner-outer-mask";
    case "md5Mask":
      return "md5-mask";
    default:
      return undefined;
  }
}

function isBuiltinSemanticType(item: SemanticTypeSetting_SemanticType) {
  return item.id.startsWith("bb.");
}

function useEscapeKey(onEscape: () => void) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onEscape();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [onEscape]);
}

function useClickOutside(
  ref: React.RefObject<HTMLElement | null>,
  onClickOutside: () => void
) {
  useEffect(() => {
    // Delay by one frame so the click that opened the popover doesn't
    // immediately trigger the outside-click handler.
    const id = requestAnimationFrame(() => {
      document.addEventListener("mousedown", handler);
    });
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClickOutside();
      }
    }
    return () => {
      cancelAnimationFrame(id);
      document.removeEventListener("mousedown", handler);
    };
  }, [ref, onClickOutside]);
}

export function SemanticTypesPage() {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();

  const subscriptionStore = useSubscriptionV1Store();

  const hasPermission = hasWorkspacePermissionV2("bb.policies.update");
  const hasSensitiveDataFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_DATA_MASKING)
  );
  const isReadonly = !hasPermission || !hasSensitiveDataFeature;

  const [items, setItems] = useState<SemanticItem[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [showTemplateDrawer, setShowTemplateDrawer] = useState(false);
  const [algorithmDrawer, setAlgorithmDrawer] = useState<{
    index: number;
    algorithm?: Algorithm;
  } | null>(null);

  // Deferred side-effect queue: state updaters push actions here,
  // and a useEffect flushes them after React commits.
  const pendingActionRef = useRef<(() => void)[]>([]);
  useEffect(() => {
    const actions = pendingActionRef.current.splice(0);
    for (const action of actions) action();
  });

  const semanticTypeSettingValue = useVueState(() => {
    const setting = settingStore.getSettingByName(
      Setting_SettingName.SEMANTIC_TYPES
    );
    return setting?.value?.value?.case === "semanticType"
      ? (setting.value.value.value.types ?? [])
      : [];
  });

  useEffect(() => {
    settingStore
      .getOrFetchSettingByName(Setting_SettingName.SEMANTIC_TYPES)
      .then(() => setLoaded(true));
  }, []);

  useEffect(() => {
    if (!loaded) return;
    setItems(
      semanticTypeSettingValue.map((st) => ({
        dirty: false,
        item: st,
        mode: "NORMAL" as const,
      }))
    );
  }, [loaded]);

  const upsertSetting = useCallback(
    async (types: SemanticTypeSetting_SemanticType[], notification: string) => {
      await settingStore.upsertSetting({
        name: Setting_SettingName.SEMANTIC_TYPES,
        value: create(SettingSettingValueSchema, {
          value: {
            case: "semanticType",
            value: { types },
          },
        }),
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: notification,
      });
    },
    []
  );

  const onAdd = useCallback(() => {
    setItems((prev) => [
      ...prev,
      {
        mode: "CREATE",
        dirty: false,
        item: create(SemanticTypeSetting_SemanticTypeSchema, {
          id: uuidv4(),
        }),
      },
    ]);
  }, []);

  const onStartEdit = useCallback((index: number) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], mode: "EDIT" };
      return next;
    });
  }, []);

  const onRemove = useCallback(
    (index: number) => {
      setItems((prev) => {
        const next = [...prev];
        const removed = next.splice(index, 1)[0];
        if (removed.mode !== "CREATE") {
          const types = next
            .filter((d) => d.mode === "NORMAL")
            .map((d) => d.item);
          pendingActionRef.current.push(() =>
            upsertSetting(types, t("common.deleted"))
          );
        }
        return next;
      });
    },
    [upsertSetting, t]
  );

  const onConfirm = useCallback(
    (index: number) => {
      setItems((prev) => {
        const next = [...prev];
        const item = next[index];
        const wasCreate = item.mode === "CREATE";
        next[index] = { ...item, dirty: false, mode: "NORMAL" };
        const types = next
          .filter((d) => d.mode === "NORMAL")
          .map((d) => d.item);
        const msg = t(wasCreate ? "common.created" : "common.updated");
        pendingActionRef.current.push(() => upsertSetting(types, msg));
        return next;
      });
    },
    [upsertSetting, t]
  );

  const onCancel = useCallback(
    (index: number) => {
      setItems((prev) => {
        const next = [...prev];
        const item = next[index];
        if (item.mode === "CREATE") {
          next.splice(index, 1);
        } else {
          const origin = semanticTypeSettingValue.find(
            (s) => s.id === item.item.id
          );
          if (origin) {
            next[index] = { item: origin, mode: "NORMAL", dirty: false };
          }
        }
        return next;
      });
    },
    [semanticTypeSettingValue]
  );

  const onInput = useCallback(
    (
      index: number,
      updater: (
        item: SemanticTypeSetting_SemanticType
      ) => SemanticTypeSetting_SemanticType
    ) => {
      setItems((prev) => {
        const next = [...prev];
        const current = next[index];
        if (!current) return prev;
        next[index] = {
          ...current,
          dirty: true,
          item: updater(current.item),
        };
        return next;
      });
    },
    []
  );

  const algorithmDrawerRef = useRef(algorithmDrawer);
  algorithmDrawerRef.current = algorithmDrawer;

  const onAlgorithmApply = useCallback(
    (algorithm: Algorithm) => {
      const drawer = algorithmDrawerRef.current;
      if (drawer === null) return;
      const { index } = drawer;
      setItems((prev) => {
        const next = [...prev];
        const current = next[index];
        if (!current) return prev;
        const updated: SemanticItem = {
          ...current,
          dirty: true,
          item: { ...current.item, algorithm },
        };
        if (updated.item.title) {
          updated.dirty = false;
          updated.mode = "NORMAL";
          next[index] = updated;
          const types = next
            .filter((d) => d.mode === "NORMAL")
            .map((d) => d.item);
          const msg = t(
            current.mode === "CREATE" ? "common.created" : "common.updated"
          );
          pendingActionRef.current.push(() => upsertSetting(types, msg));
        } else {
          next[index] = updated;
        }
        return next;
      });
      setAlgorithmDrawer(null);
    },
    [upsertSetting, t]
  );

  const onOpenAlgorithmDrawer = useCallback(
    (index: number, algorithm?: Algorithm) => {
      setAlgorithmDrawer({ index, algorithm });
    },
    []
  );

  const onTemplateApply = useCallback(
    (template: SemanticTypeSetting_SemanticType) => {
      setItems((prev) => {
        if (prev.find((item) => item.item.id === template.id)) {
          pendingActionRef.current.push(() =>
            pushNotification({
              module: "bytebase",
              style: "INFO",
              title: t(
                "settings.sensitive-data.semantic-types.template.duplicate-warning",
                { title: template.title }
              ),
            })
          );
          return prev;
        }
        const newItem: SemanticItem = {
          dirty: false,
          mode: "NORMAL",
          item: create(SemanticTypeSetting_SemanticTypeSchema, {
            ...template,
          }),
        };
        const next = [...prev, newItem];
        const types = next
          .filter((d) => d.mode === "NORMAL")
          .map((d) => d.item);
        pendingActionRef.current.push(() =>
          upsertSetting(types, t("common.created"))
        );
        return next;
      });
      setShowTemplateDrawer(false);
    },
    [upsertSetting, t]
  );

  const isConfirmDisabled = (data: SemanticItem): boolean => {
    if (!data.item.title) return true;
    if (data.mode === "EDIT" && !data.dirty) return true;
    return false;
  };

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-4">
      <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />
      <div className="flex justify-end">
        <div className="flex items-center gap-x-2">
          <Button
            variant="outline"
            disabled={isReadonly}
            onClick={() => setShowTemplateDrawer(true)}
          >
            {t("settings.sensitive-data.semantic-types.use-predefined-type")}
          </Button>
          <Button disabled={isReadonly} onClick={onAdd}>
            <Plus className="h-4 w-4" />
            {t("common.add")}
          </Button>
        </div>
      </div>

      <p className="text-sm text-control-placeholder">
        {t("settings.sensitive-data.semantic-types.label")}
      </p>

      <div className="border border-control-border rounded-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-gray-50 border-b border-control-border">
              <th className="px-3 py-2 font-medium w-20 text-center">
                {t("settings.sensitive-data.semantic-types.table.icon")}
              </th>
              <th className="px-3 py-2 text-left font-medium w-36">ID</th>
              <th className="px-3 py-2 text-left font-medium">
                {t(
                  "settings.sensitive-data.semantic-types.table.semantic-type"
                )}
              </th>
              <th className="px-3 py-2 text-left font-medium w-48">
                {t("settings.sensitive-data.semantic-types.table.description")}
              </th>
              <th className="px-3 py-2 text-left font-medium">
                {t(
                  "settings.sensitive-data.semantic-types.table.masking-algorithm"
                )}
              </th>
              {!isReadonly && (
                <th className="px-3 py-2 text-right font-medium w-28">
                  {t("common.edit")}
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {items.map((row, index) => (
              <SemanticTypeRow
                key={row.item.id}
                row={row}
                index={index}
                readonly={isReadonly}
                isConfirmDisabled={isConfirmDisabled}
                onInput={onInput}
                onRemove={onRemove}
                onConfirm={onConfirm}
                onCancel={onCancel}
                onStartEdit={onStartEdit}
                onOpenAlgorithmDrawer={onOpenAlgorithmDrawer}
              />
            ))}
            {items.length === 0 && (
              <tr>
                <td
                  colSpan={isReadonly ? 5 : 6}
                  className="px-3 py-8 text-center text-control-placeholder"
                >
                  {t("common.no-data")}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {showTemplateDrawer && (
        <SemanticTemplateDrawer
          onApply={onTemplateApply}
          onDismiss={() => setShowTemplateDrawer(false)}
        />
      )}

      {algorithmDrawer !== null && (
        <MaskingAlgorithmDrawer
          algorithm={algorithmDrawer.algorithm}
          onApply={onAlgorithmApply}
          onDismiss={() => setAlgorithmDrawer(null)}
        />
      )}
    </div>
  );
}

// --- MaskingAlgorithmDrawer (React) ---

interface MaskingAlgorithmDrawerProps {
  algorithm?: Algorithm;
  onApply: (algorithm: Algorithm) => void;
  onDismiss: () => void;
}

interface RangeMaskSlice {
  start: number;
  end: number;
  substitution: string;
}

function MaskingAlgorithmDrawer({
  algorithm,
  onApply,
  onDismiss,
}: MaskingAlgorithmDrawerProps) {
  const { t } = useTranslation();
  useEscapeKey(onDismiss);

  const initialType = getMaskingType(algorithm) ?? "full-mask";
  const [maskingType, setMaskingType] = useState<MaskingType>(initialType);
  const [fullMaskSubstitution, setFullMaskSubstitution] = useState(
    algorithm?.mask?.case === "fullMask"
      ? (algorithm.mask.value.substitution ?? "")
      : ""
  );
  const [rangeMaskSlices, setRangeMaskSlices] = useState<RangeMaskSlice[]>(
    algorithm?.mask?.case === "rangeMask"
      ? algorithm.mask.value.slices.map((s) => ({
          start: s.start,
          end: s.end,
          substitution: s.substitution,
        }))
      : [{ start: 0, end: 1, substitution: "*" }]
  );
  const [md5Salt, setMd5Salt] = useState(
    algorithm?.mask?.case === "md5Mask" ? (algorithm.mask.value.salt ?? "") : ""
  );
  const [innerOuterType, setInnerOuterType] = useState(
    algorithm?.mask?.case === "innerOuterMask"
      ? algorithm.mask.value.type
      : Algorithm_InnerOuterMask_MaskType.INNER
  );
  const [innerOuterPrefix, setInnerOuterPrefix] = useState(
    algorithm?.mask?.case === "innerOuterMask"
      ? algorithm.mask.value.prefixLen
      : 0
  );
  const [innerOuterSuffix, setInnerOuterSuffix] = useState(
    algorithm?.mask?.case === "innerOuterMask"
      ? algorithm.mask.value.suffixLen
      : 0
  );
  const [innerOuterSubstitution, setInnerOuterSubstitution] = useState(
    algorithm?.mask?.case === "innerOuterMask"
      ? (algorithm.mask.value.substitution ?? "*")
      : "*"
  );

  const maskingTypeOptions: { value: MaskingType; label: string }[] = [
    {
      value: "full-mask",
      label: t("settings.sensitive-data.algorithms.full-mask.self"),
    },
    {
      value: "range-mask",
      label: t("settings.sensitive-data.algorithms.range-mask.self"),
    },
    {
      value: "md5-mask",
      label: t("settings.sensitive-data.algorithms.md5-mask.self"),
    },
    {
      value: "inner-outer-mask",
      label: t("settings.sensitive-data.algorithms.inner-outer-mask.self"),
    },
  ];

  const onMaskingTypeChange = (type: MaskingType) => {
    setMaskingType(type);
    if (type === "full-mask") setFullMaskSubstitution("");
    if (type === "range-mask")
      setRangeMaskSlices([{ start: 0, end: 1, substitution: "*" }]);
    if (type === "md5-mask") setMd5Salt("");
    if (type === "inner-outer-mask") {
      setInnerOuterType(Algorithm_InnerOuterMask_MaskType.INNER);
      setInnerOuterPrefix(0);
      setInnerOuterSuffix(0);
      setInnerOuterSubstitution("*");
    }
  };

  const rangeMaskErrorMessage = useMemo(() => {
    if (rangeMaskSlices.length === 0) {
      return t("settings.sensitive-data.algorithms.error.slice-required");
    }
    for (let i = 0; i < rangeMaskSlices.length; i++) {
      const slice = rangeMaskSlices[i];
      if (Number.isNaN(slice.start) || Number.isNaN(slice.end)) {
        return t(
          "settings.sensitive-data.algorithms.error.slice-invalid-number"
        );
      }
      for (let j = 0; j < i; j++) {
        const pre = rangeMaskSlices[j];
        if (!(slice.start >= pre.end || pre.start >= slice.end)) {
          return t("settings.sensitive-data.algorithms.error.slice-overlap");
        }
      }
      if (!slice.substitution) {
        return t(
          "settings.sensitive-data.algorithms.error.substitution-required"
        );
      }
      if (slice.substitution.length > 16) {
        return t(
          "settings.sensitive-data.algorithms.error.substitution-length"
        );
      }
    }
    return "";
  }, [rangeMaskSlices, t]);

  const errorMessage = useMemo(() => {
    switch (maskingType) {
      case "full-mask":
        if (!fullMaskSubstitution)
          return t(
            "settings.sensitive-data.algorithms.error.substitution-required"
          );
        if (fullMaskSubstitution.length > 16)
          return t(
            "settings.sensitive-data.algorithms.error.substitution-length"
          );
        return "";
      case "md5-mask":
        if (!md5Salt)
          return t("settings.sensitive-data.algorithms.error.salt-required");
        return "";
      case "range-mask":
        return rangeMaskErrorMessage;
      case "inner-outer-mask":
        if (!innerOuterSubstitution)
          return t(
            "settings.sensitive-data.algorithms.error.substitution-required"
          );
        if (innerOuterSubstitution.length > 16)
          return t(
            "settings.sensitive-data.algorithms.error.substitution-length"
          );
        return "";
    }
    return "";
  }, [
    maskingType,
    fullMaskSubstitution,
    md5Salt,
    rangeMaskErrorMessage,
    innerOuterSubstitution,
    t,
  ]);

  const buildAlgorithm = (): Algorithm => {
    switch (maskingType) {
      case "full-mask":
        return create(AlgorithmSchema, {
          mask: {
            case: "fullMask",
            value: create(FullMaskSchema, {
              substitution: fullMaskSubstitution,
            }),
          },
        });
      case "range-mask":
        return create(AlgorithmSchema, {
          mask: {
            case: "rangeMask",
            value: create(RangeMaskSchema, {
              slices: rangeMaskSlices.map((s) =>
                create(Algorithm_RangeMask_SliceSchema, s)
              ),
            }),
          },
        });
      case "md5-mask":
        return create(AlgorithmSchema, {
          mask: {
            case: "md5Mask",
            value: create(MD5MaskSchema, { salt: md5Salt }),
          },
        });
      case "inner-outer-mask":
        return create(AlgorithmSchema, {
          mask: {
            case: "innerOuterMask",
            value: create(Algorithm_InnerOuterMaskSchema, {
              type: innerOuterType,
              prefixLen: innerOuterPrefix,
              suffixLen: innerOuterSuffix,
              substitution: innerOuterSubstitution,
            }),
          },
        });
    }
  };

  const updateSlice = (index: number, patch: Partial<RangeMaskSlice>) => {
    setRangeMaskSlices((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], ...patch };
      return next;
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/40" onClick={onDismiss} />
      <div className="relative w-[40rem] max-w-[calc(100vw-5rem)] bg-white shadow-xl flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-medium">
            {algorithm
              ? t("settings.sensitive-data.algorithms.edit")
              : t("settings.sensitive-data.algorithms.add")}
          </h2>
          <button
            className="p-1 rounded hover:bg-gray-200 text-gray-500"
            onClick={onDismiss}
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-6">
          {/* Masking type selector */}
          <div className="mb-6">
            <label className="text-sm font-medium">
              {t("settings.sensitive-data.algorithms.table.masking-type")}
              <span className="text-red-500 ml-0.5">*</span>
            </label>
            <div className="grid grid-cols-3 gap-2 mt-2">
              {maskingTypeOptions.map((opt) => (
                <button
                  key={opt.value}
                  className={`px-3 py-2 text-sm border rounded-md transition-colors ${
                    maskingType === opt.value
                      ? "border-accent bg-accent/10 text-accent"
                      : "border-control-border hover:bg-gray-50"
                  }`}
                  onClick={() => onMaskingTypeChange(opt.value)}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <div className="border-t border-control-border pt-6 flex flex-col gap-y-6">
            {maskingType === "full-mask" && (
              <div>
                <label className="text-sm font-medium">
                  {t(
                    "settings.sensitive-data.algorithms.full-mask.substitution"
                  )}
                  <span className="text-red-500 ml-0.5">*</span>
                </label>
                <p className="text-sm text-control-placeholder mt-1">
                  {t(
                    "settings.sensitive-data.algorithms.full-mask.substitution-label"
                  )}
                </p>
                <Input
                  value={fullMaskSubstitution}
                  className="mt-2"
                  placeholder={t(
                    "settings.sensitive-data.algorithms.full-mask.substitution"
                  )}
                  onChange={(e) => setFullMaskSubstitution(e.target.value)}
                />
              </div>
            )}

            {maskingType === "range-mask" && (
              <>
                <p className="text-sm text-control-placeholder">
                  {t("settings.sensitive-data.algorithms.range-mask.label")}
                </p>
                {rangeMaskSlices.map((slice, i) => (
                  <div key={i} className="flex gap-x-2 items-end">
                    <div className="flex flex-col gap-y-1">
                      <label className="text-sm font-medium">
                        {t(
                          "settings.sensitive-data.algorithms.range-mask.slice-start"
                        )}
                        <span className="text-red-500 ml-0.5">*</span>
                      </label>
                      <Input
                        type="number"
                        value={slice.start}
                        className="w-20"
                        onChange={(e) =>
                          updateSlice(i, {
                            start: Number(e.target.value),
                            end: Math.max(
                              Number(e.target.value) + 1,
                              slice.end
                            ),
                          })
                        }
                      />
                    </div>
                    <div className="flex flex-col gap-y-1">
                      <label className="text-sm font-medium">
                        {t(
                          "settings.sensitive-data.algorithms.range-mask.slice-end"
                        )}
                        <span className="text-red-500 ml-0.5">*</span>
                      </label>
                      <Input
                        type="number"
                        value={slice.end}
                        className="w-20"
                        onChange={(e) =>
                          updateSlice(i, { end: Number(e.target.value) })
                        }
                      />
                    </div>
                    <div className="flex-1 flex flex-col gap-y-1">
                      <label className="text-sm font-medium">
                        {t(
                          "settings.sensitive-data.algorithms.range-mask.substitution"
                        )}
                        <span className="text-red-500 ml-0.5">*</span>
                      </label>
                      <Input
                        value={slice.substitution}
                        onChange={(e) =>
                          updateSlice(i, { substitution: e.target.value })
                        }
                      />
                    </div>
                    <button
                      className="p-1 rounded hover:bg-red-100 text-red-500 mb-0.5"
                      onClick={() =>
                        setRangeMaskSlices((prev) =>
                          prev.filter((_, idx) => idx !== i)
                        )
                      }
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
                {rangeMaskErrorMessage && (
                  <p className="text-red-600 text-sm">
                    {rangeMaskErrorMessage}
                  </p>
                )}
                <Button
                  variant="outline"
                  className="ml-auto"
                  onClick={() =>
                    setRangeMaskSlices((prev) => {
                      const last = prev[prev.length - 1];
                      return [
                        ...prev,
                        {
                          start: (last?.start ?? -1) + 1,
                          end: (last?.end ?? 0) + 1,
                          substitution: "*",
                        },
                      ];
                    })
                  }
                >
                  {t("common.add")}
                </Button>
              </>
            )}

            {maskingType === "md5-mask" && (
              <div>
                <label className="text-sm font-medium">
                  {t("settings.sensitive-data.algorithms.md5-mask.salt")}
                  <span className="text-red-500 ml-0.5">*</span>
                </label>
                <p className="text-sm text-control-placeholder mt-1">
                  {t("settings.sensitive-data.algorithms.md5-mask.salt-label")}
                </p>
                <Input
                  value={md5Salt}
                  className="mt-2"
                  placeholder={t(
                    "settings.sensitive-data.algorithms.md5-mask.salt"
                  )}
                  onChange={(e) => setMd5Salt(e.target.value)}
                />
              </div>
            )}

            {maskingType === "inner-outer-mask" && (
              <>
                <div>
                  <label className="text-sm font-medium">
                    {t(
                      "settings.sensitive-data.algorithms.inner-outer-mask.type"
                    )}
                    <span className="text-red-500 ml-0.5">*</span>
                  </label>
                  <p className="text-sm text-control-placeholder mt-1">
                    {innerOuterType === Algorithm_InnerOuterMask_MaskType.INNER
                      ? t(
                          "settings.sensitive-data.algorithms.inner-outer-mask.inner-label"
                        )
                      : t(
                          "settings.sensitive-data.algorithms.inner-outer-mask.outer-label"
                        )}
                  </p>
                  <div className="flex gap-x-4 mt-2">
                    <label className="flex items-center gap-x-2 cursor-pointer">
                      <input
                        type="radio"
                        name="innerOuterType"
                        checked={
                          innerOuterType ===
                          Algorithm_InnerOuterMask_MaskType.INNER
                        }
                        onChange={() =>
                          setInnerOuterType(
                            Algorithm_InnerOuterMask_MaskType.INNER
                          )
                        }
                      />
                      {t(
                        "settings.sensitive-data.algorithms.inner-outer-mask.inner-mask"
                      )}
                    </label>
                    <label className="flex items-center gap-x-2 cursor-pointer">
                      <input
                        type="radio"
                        name="innerOuterType"
                        checked={
                          innerOuterType ===
                          Algorithm_InnerOuterMask_MaskType.OUTER
                        }
                        onChange={() =>
                          setInnerOuterType(
                            Algorithm_InnerOuterMask_MaskType.OUTER
                          )
                        }
                      />
                      {t(
                        "settings.sensitive-data.algorithms.inner-outer-mask.outer-mask"
                      )}
                    </label>
                  </div>
                </div>
                <div className="flex gap-x-2 items-end">
                  <div className="flex flex-col gap-y-1">
                    <label className="text-sm font-medium">
                      {t(
                        "settings.sensitive-data.algorithms.inner-outer-mask.prefix-length"
                      )}
                      <span className="text-red-500 ml-0.5">*</span>
                    </label>
                    <Input
                      type="number"
                      value={innerOuterPrefix}
                      className="w-24"
                      onChange={(e) => {
                        const val = Number(e.target.value);
                        if (!Number.isNaN(val) && val >= 0)
                          setInnerOuterPrefix(val);
                      }}
                    />
                  </div>
                  <div className="flex flex-col gap-y-1">
                    <label className="text-sm font-medium">
                      {t(
                        "settings.sensitive-data.algorithms.inner-outer-mask.suffix-length"
                      )}
                      <span className="text-red-500 ml-0.5">*</span>
                    </label>
                    <Input
                      type="number"
                      value={innerOuterSuffix}
                      className="w-24"
                      onChange={(e) => {
                        const val = Number(e.target.value);
                        if (!Number.isNaN(val) && val >= 0)
                          setInnerOuterSuffix(val);
                      }}
                    />
                  </div>
                  <div className="flex-1 flex flex-col gap-y-1">
                    <label className="text-sm font-medium">
                      {t(
                        "settings.sensitive-data.algorithms.range-mask.substitution"
                      )}
                      <span className="text-red-500 ml-0.5">*</span>
                    </label>
                    <Input
                      value={innerOuterSubstitution}
                      onChange={(e) =>
                        setInnerOuterSubstitution(e.target.value)
                      }
                    />
                  </div>
                </div>
              </>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-x-2 px-6 py-4 border-t border-control-border">
          <Button variant="outline" onClick={onDismiss}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!!errorMessage}
            onClick={() => onApply(buildAlgorithm())}
            title={errorMessage || undefined}
          >
            {t("common.confirm")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// --- SemanticTemplateDrawer (React) ---

interface SemanticTemplateDrawerProps {
  onApply: (template: SemanticTypeSetting_SemanticType) => void;
  onDismiss: () => void;
}

function SemanticTemplateDrawer({
  onApply,
  onDismiss,
}: SemanticTemplateDrawerProps) {
  const { t } = useTranslation();
  useEscapeKey(onDismiss);
  const templates = useMemo(() => getSemanticTemplateList(), []);

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/40" onClick={onDismiss} />
      <div className="relative w-2xl max-w-[100vw] bg-white shadow-xl flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-medium">
            {t("settings.sensitive-data.semantic-types.add-from-template")}
          </h2>
          <button
            className="p-1 rounded hover:bg-gray-200 text-gray-500"
            onClick={onDismiss}
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-6">
          <p className="text-sm text-control-placeholder mb-4">
            {t("settings.sensitive-data.semantic-types.template.description")}
          </p>
          <div className="border border-control-border rounded-sm overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-50 border-b border-control-border">
                  <th className="px-3 py-2 text-left font-medium">ID</th>
                  <th className="px-3 py-2 text-left font-medium">
                    {t(
                      "settings.sensitive-data.semantic-types.table.semantic-type"
                    )}
                  </th>
                  <th className="px-3 py-2 text-left font-medium">
                    {t(
                      "settings.sensitive-data.semantic-types.table.description"
                    )}
                  </th>
                  <th className="px-3 py-2 text-left font-medium">
                    {t(
                      "settings.sensitive-data.semantic-types.table.masking-algorithm"
                    )}
                  </th>
                </tr>
              </thead>
              <tbody>
                {templates.map((template) => {
                  const key = template.id.split(".").join("-");
                  return (
                    <tr
                      key={template.id}
                      className="border-b border-control-border last:border-b-0 hover:bg-gray-50 cursor-pointer"
                      onClick={() => onApply(template)}
                    >
                      <td className="px-3 py-2">{template.id}</td>
                      <td className="px-3 py-2">{template.title}</td>
                      <td className="px-3 py-2">{template.description}</td>
                      <td className="px-3 py-2">
                        <div className="flex items-center gap-x-1">
                          <span>
                            {t(
                              `dynamic.settings.sensitive-data.semantic-types.template.${key}.title`
                            )}
                          </span>
                          <span
                            className="text-gray-400 cursor-help"
                            title={t(
                              `dynamic.settings.sensitive-data.semantic-types.template.${key}.algorithm.description`
                            )}
                          >
                            <Info className="w-4 h-4" />
                          </span>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
        <div className="flex justify-end px-6 py-4 border-t border-control-border">
          <Button variant="outline" onClick={onDismiss}>
            {t("common.cancel")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// --- SemanticTypeRow ---

interface SemanticTypeRowProps {
  row: SemanticItem;
  index: number;
  readonly: boolean;
  isConfirmDisabled: (data: SemanticItem) => boolean;
  onInput: (
    index: number,
    updater: (
      item: SemanticTypeSetting_SemanticType
    ) => SemanticTypeSetting_SemanticType
  ) => void;
  onRemove: (index: number) => void;
  onConfirm: (index: number) => void;
  onCancel: (index: number) => void;
  onStartEdit: (index: number) => void;
  onOpenAlgorithmDrawer: (index: number, algorithm?: Algorithm) => void;
}

function SemanticTypeRow({
  row,
  index,
  readonly,
  isConfirmDisabled,
  onInput,
  onRemove,
  onConfirm,
  onCancel,
  onStartEdit,
  onOpenAlgorithmDrawer,
}: SemanticTypeRowProps) {
  const { t } = useTranslation();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const isBuiltin = isBuiltinSemanticType(row.item);
  const isItemReadonly = readonly || isBuiltin;
  const isEditing = row.mode !== "NORMAL";

  return (
    <tr className="border-b border-control-border last:border-b-0 even:bg-gray-50/50">
      <td className="px-3 py-2 text-center">
        {isEditing && !isItemReadonly ? (
          <IconPicker
            value={row.item.icon ?? ""}
            onChange={(icon) => onInput(index, (item) => ({ ...item, icon }))}
          />
        ) : row.item.icon ? (
          <div className="flex items-center justify-center">
            <img
              src={row.item.icon}
              className="w-6 h-6 object-contain"
              alt=""
            />
          </div>
        ) : (
          <span className="text-gray-400">-</span>
        )}
      </td>
      <td className="px-3 py-2 truncate max-w-36" title={row.item.id}>
        {row.item.id}
      </td>
      <td className="px-3 py-2">
        {isEditing ? (
          <Input
            value={row.item.title}
            className="h-8"
            placeholder={t(
              "settings.sensitive-data.semantic-types.table.semantic-type"
            )}
            onChange={(e) =>
              onInput(index, (item) => ({ ...item, title: e.target.value }))
            }
          />
        ) : (
          <span className="truncate">{row.item.title}</span>
        )}
      </td>
      <td className="px-3 py-2">
        {isEditing ? (
          <Input
            value={row.item.description}
            className="h-8"
            placeholder={t(
              "settings.sensitive-data.semantic-types.table.description"
            )}
            onChange={(e) =>
              onInput(index, (item) => ({
                ...item,
                description: e.target.value,
              }))
            }
          />
        ) : (
          <span className="truncate">{row.item.description}</span>
        )}
      </td>
      <td className="px-3 py-2">
        <div className="flex items-center gap-x-1">
          {isBuiltin ? (
            <>
              <span>
                {t(
                  `dynamic.settings.sensitive-data.semantic-types.template.${row.item.id.split(".").join("-")}.title`
                )}
              </span>
              <span
                className="text-gray-400 cursor-help"
                title={t(
                  `dynamic.settings.sensitive-data.semantic-types.template.${row.item.id.split(".").join("-")}.algorithm.description`
                )}
              >
                <Info className="w-4 h-4" />
              </span>
            </>
          ) : (
            <span>
              {getMaskingType(row.item.algorithm)
                ? t(
                    `settings.sensitive-data.algorithms.${getMaskingType(row.item.algorithm)?.toLowerCase()}.self`
                  )
                : "N/A"}
            </span>
          )}
          {!isItemReadonly && (
            <button
              className="p-1 rounded hover:bg-gray-200 text-gray-500"
              onClick={() => {
                const algo = getMaskingType(row.item.algorithm)
                  ? row.item.algorithm
                  : undefined;
                onOpenAlgorithmDrawer(index, algo);
              }}
            >
              <Pencil className="w-4 h-4" />
            </button>
          )}
        </div>
      </td>
      {!readonly && (
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-x-1">
            {isBuiltin ? (
              <DeleteConfirmButton
                show={showDeleteConfirm}
                onShowChange={setShowDeleteConfirm}
                message={t(
                  "settings.sensitive-data.semantic-types.table.delete"
                )}
                onConfirm={() => onRemove(index)}
              />
            ) : (
              <>
                {isEditing && (
                  <button
                    className="p-1 rounded hover:bg-gray-200 text-gray-500"
                    onClick={() => onCancel(index)}
                  >
                    <Undo2 className="w-4 h-4" />
                  </button>
                )}
                {row.mode === "EDIT" && (
                  <DeleteConfirmButton
                    show={showDeleteConfirm}
                    onShowChange={setShowDeleteConfirm}
                    message={t(
                      "settings.sensitive-data.semantic-types.table.delete"
                    )}
                    onConfirm={() => onRemove(index)}
                  />
                )}
                {isEditing && (
                  <button
                    className="p-1 rounded hover:bg-accent/10 text-accent disabled:opacity-50 disabled:cursor-not-allowed"
                    disabled={isConfirmDisabled(row)}
                    onClick={() => onConfirm(index)}
                  >
                    <Check className="w-4 h-4" />
                  </button>
                )}
                {row.mode === "NORMAL" && (
                  <button
                    className="p-1 rounded hover:bg-gray-200 text-gray-500"
                    onClick={() => onStartEdit(index)}
                  >
                    <Pencil className="w-4 h-4" />
                  </button>
                )}
              </>
            )}
          </div>
        </td>
      )}
    </tr>
  );
}

// --- DeleteConfirmButton ---

// --- IconPicker ---

const SUPPORTED_IMAGE_EXTENSIONS = [".jpg", ".jpeg", ".png", ".webp", ".svg"];
const MAX_FILE_SIZE_BYTES = 2 * 1024 * 1024; // 2 MiB

interface IconPickerProps {
  value: string;
  onChange: (base64: string) => void;
}

function IconPicker({ value, onChange }: IconPickerProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [tempValue, setTempValue] = useState(value);
  const popoverRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const dismiss = useCallback(() => setOpen(false), []);
  useClickOutside(popoverRef, dismiss);

  const handleOpen = () => {
    setTempValue(value);
    setOpen(true);
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > MAX_FILE_SIZE_BYTES) return;
    const reader = new FileReader();
    reader.onload = () => setTempValue(reader.result as string);
    reader.readAsDataURL(file);
    // Reset input so the same file can be re-selected
    e.target.value = "";
  };

  return (
    <div className="relative flex items-center justify-center" ref={popoverRef}>
      {value ? (
        <div className="flex items-center gap-1">
          <img src={value} className="w-6 h-6 object-contain" alt="" />
          <button
            className="p-0.5 rounded hover:bg-gray-200 text-gray-500"
            onClick={handleOpen}
          >
            <Pencil className="w-3 h-3" />
          </button>
        </div>
      ) : (
        <button
          className="p-1 rounded hover:bg-gray-200 text-gray-500"
          onClick={handleOpen}
        >
          <Pencil className="w-4 h-4" />
        </button>
      )}
      {open && (
        <div className="absolute left-0 top-full z-20 mt-1 bg-white border border-control-border rounded-sm shadow-lg p-3">
          <div
            className="w-48 h-48 flex justify-center items-center border border-dashed border-gray-300 rounded-sm relative cursor-pointer"
            onClick={() => fileInputRef.current?.click()}
          >
            {tempValue ? (
              <div
                className="w-1/3 h-1/3 bg-no-repeat bg-contain bg-center rounded-md"
                style={{ backgroundImage: `url(${tempValue})` }}
              />
            ) : (
              <span className="text-sm text-gray-400">
                {t("common.upload")}
              </span>
            )}
            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              accept={SUPPORTED_IMAGE_EXTENSIONS.join(",")}
              onChange={handleFileSelect}
            />
          </div>
          <div className="flex justify-end gap-x-2 mt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setTempValue("")}
            >
              {t("common.clear")}
            </Button>
            <Button variant="outline" size="sm" onClick={() => setOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              size="sm"
              onClick={() => {
                onChange(tempValue);
                setOpen(false);
              }}
            >
              {t("common.confirm")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// --- DeleteConfirmButton ---

interface DeleteConfirmButtonProps {
  show: boolean;
  onShowChange: (show: boolean) => void;
  message: string;
  onConfirm: () => void;
}

function DeleteConfirmButton({
  show,
  onShowChange,
  message,
  onConfirm,
}: DeleteConfirmButtonProps) {
  const { t } = useTranslation();
  const popoverRef = useRef<HTMLDivElement>(null);
  const dismiss = useCallback(() => onShowChange(false), [onShowChange]);
  useClickOutside(popoverRef, dismiss);

  return (
    <div className="relative" ref={popoverRef}>
      <button
        className="p-1 rounded hover:bg-red-100 text-red-500"
        onClick={() => onShowChange(!show)}
      >
        <Trash2 className="w-4 h-4" />
      </button>
      {show && (
        <div className="absolute right-0 top-full z-10 mt-1 bg-white border border-control-border rounded-sm shadow-lg p-3 whitespace-nowrap">
          <p className="text-sm mb-2">{message}</p>
          <div className="flex justify-end gap-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onShowChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                onConfirm();
                onShowChange(false);
              }}
            >
              {t("common.confirm")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
