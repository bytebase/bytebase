import { Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { MAX_LABEL_VALUE_LENGTH } from "@/utils";

export interface LabelListEditorItem {
  key: string;
  value: string;
  message?: string;
  allowEmpty?: boolean;
}

export interface LabelListEditorFlattenedError {
  key: string;
  errors: string[];
}

export interface LabelListEditorProps {
  kvList: LabelListEditorItem[];
  onChange: (list: LabelListEditorItem[]) => void;
  readonly: boolean;
  showErrors?: boolean;
  onErrorsChange?: (errors: string[]) => void;
  onFlattenErrorsChange?: (errors: LabelListEditorFlattenedError[]) => void;
}

export function LabelListEditor({
  kvList,
  onChange,
  readonly,
  showErrors = true,
  onErrorsChange,
  onFlattenErrorsChange,
}: LabelListEditorProps) {
  const { t } = useTranslation();

  const errorList = useMemo(() => {
    return kvList.map((kv) => {
      const errors = { key: [] as string[], value: [] as string[] };
      if (!kv.key) {
        errors.key.push(t("label.error.key-necessary"));
      } else if (kvList.filter((item) => item.key === kv.key).length > 1) {
        errors.key.push(t("label.error.key-duplicated"));
      }
      if (!kv.value) {
        if (kv.allowEmpty) {
          return errors;
        }
        errors.value.push(t("label.error.value-necessary"));
      } else if (kv.value.length > MAX_LABEL_VALUE_LENGTH) {
        errors.value.push(
          t("label.error.max-value-length-exceeded", {
            length: MAX_LABEL_VALUE_LENGTH,
          })
        );
      }
      return errors;
    });
  }, [kvList, t]);

  const visibleErrorList = useMemo(
    () =>
      showErrors
        ? errorList
        : kvList.map(() => ({ key: [] as string[], value: [] as string[] })),
    [errorList, kvList, showErrors]
  );

  const flattenedErrors = useMemo(() => {
    return kvList.reduce<LabelListEditorFlattenedError[]>((list, kv, index) => {
      const errors = visibleErrorList[index] ?? { key: [], value: [] };
      const combinedErrors = [...errors.key, ...errors.value];
      if (combinedErrors.length > 0) {
        list.push({
          key: kv.key,
          errors: combinedErrors,
        });
      }
      return list;
    }, []);
  }, [kvList, visibleErrorList]);

  const flatErrors = useMemo(() => {
    return flattenedErrors.flatMap((item) => item.errors);
  }, [flattenedErrors]);

  useEffect(() => {
    onErrorsChange?.(flatErrors);
    onFlattenErrorsChange?.(flattenedErrors);
  }, [flatErrors, flattenedErrors, onErrorsChange, onFlattenErrorsChange]);

  const allowAddLabel = flatErrors.length === 0;

  const updateKey = (index: number, key: string) => {
    onChange(kvList.map((kv, i) => (i === index ? { ...kv, key } : kv)));
  };

  const updateValue = (index: number, value: string) => {
    onChange(kvList.map((kv, i) => (i === index ? { ...kv, value } : kv)));
  };

  const handleRemove = (index: number) => {
    onChange(kvList.filter((_, i) => i !== index));
  };

  const handleAdd = () => {
    onChange([...kvList, { key: "", value: "" }]);
  };

  return (
    <div className="flex flex-col gap-y-2">
      <div className="flex flex-wrap gap-x-2 gap-y-2">
        {kvList.map((kv, index) => {
          const errors = visibleErrorList[index] ?? { key: [], value: [] };
          const combinedErrors = [...errors.key, ...errors.value];

          return (
            <div key={index} className="flex flex-col gap-y-1">
              <div className="text-sm flex gap-x-2">
                <div className="flex flex-col">
                  <span className="mb-1 text-xs font-medium">
                    {t("common.key")} {index + 1}
                  </span>
                  {readonly ? (
                    <span className="leading-[34px]">{kv.key}</span>
                  ) : (
                    <Input
                      value={kv.key}
                      placeholder={t("setting.label.key-placeholder")}
                      className={errors.key.length > 0 ? "border-error" : ""}
                      onChange={(event) => updateKey(index, event.target.value)}
                    />
                  )}
                </div>
                <div className="flex flex-col">
                  <span className="mb-1 text-xs font-medium">
                    {t("common.value")} {index + 1}
                  </span>
                  <div className="flex items-center gap-x-2">
                    {readonly ? (
                      <span className="leading-[34px]">
                        {kv.value || (
                          <span className="text-control-placeholder">
                            {t("label.empty-label-value")}
                          </span>
                        )}
                      </span>
                    ) : (
                      <Input
                        value={kv.value}
                        placeholder={t("setting.label.value-placeholder")}
                        className={
                          errors.value.length > 0 ? "border-error" : ""
                        }
                        onChange={(event) =>
                          updateValue(index, event.target.value)
                        }
                      />
                    )}
                    <button
                      type="button"
                      aria-label={t("common.delete")}
                      className={`ml-1 ${readonly ? "invisible" : "visible"} text-control-light hover:text-error`}
                      onClick={() => handleRemove(index)}
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                  {kv.message && (
                    <div className="textinfolabel">{kv.message}</div>
                  )}
                </div>
              </div>
              {combinedErrors.length > 0 && (
                <ul className="list-disc list-outside pl-4 text-xs text-error">
                  {combinedErrors.map((error) => (
                    <li key={error}>{error}</li>
                  ))}
                </ul>
              )}
            </div>
          );
        })}
      </div>
      <div>
        <Button
          variant="outline"
          size="sm"
          disabled={readonly || !allowAddLabel}
          onClick={handleAdd}
        >
          <Plus className="mr-1 h-4 w-4" />
          {t("label.add-label")}
        </Button>
      </div>
    </div>
  );
}
