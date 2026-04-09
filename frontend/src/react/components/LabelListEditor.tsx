import { Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";

const MAX_LABEL_VALUE_LENGTH = 256;

export function LabelListEditor({
  kvList,
  onChange,
  readonly,
  showErrors = true,
  onErrorsChange,
}: {
  kvList: { key: string; value: string }[];
  onChange: (list: { key: string; value: string }[]) => void;
  readonly: boolean;
  showErrors?: boolean;
  onErrorsChange?: (errors: string[]) => void;
}) {
  const { t } = useTranslation();

  const errorList = useMemo(() => {
    return kvList.map((kv) => {
      const errors = { key: [] as string[], value: [] as string[] };
      if (!kv.key) {
        errors.key.push(t("label.error.key-necessary"));
      } else if (kvList.filter((k) => k.key === kv.key).length > 1) {
        errors.key.push(t("label.error.key-duplicated"));
      }
      if (!kv.value) {
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

  const flatErrors = useMemo(() => {
    return errorList.flatMap((e) => [...e.key, ...e.value]);
  }, [errorList]);

  const hasErrors = flatErrors.length > 0;

  useEffect(() => {
    onErrorsChange?.(flatErrors);
  }, [flatErrors, onErrorsChange]);

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
      {kvList.map((kv, index) => {
        const errors = showErrors
          ? (errorList[index] ?? { key: [], value: [] })
          : { key: [], value: [] };
        const combinedErrors = [...errors.key, ...errors.value];
        return (
          <div key={index} className="flex flex-col gap-y-1">
            <div className="flex items-center gap-x-2">
              {readonly ? (
                <span className="flex-1 text-sm">{kv.key}</span>
              ) : (
                <Input
                  className={`flex-1 ${errors.key.length > 0 ? "border-error" : ""}`}
                  value={kv.key}
                  placeholder={t("setting.label.key-placeholder")}
                  onChange={(e) => updateKey(index, e.target.value)}
                />
              )}
              {readonly ? (
                <span className="flex-1 text-sm">
                  {kv.value || (
                    <span className="text-control-placeholder">
                      {t("label.empty-label-value")}
                    </span>
                  )}
                </span>
              ) : (
                <Input
                  className={`flex-1 ${errors.value.length > 0 ? "border-error" : ""}`}
                  value={kv.value}
                  placeholder={t("setting.label.value-placeholder")}
                  onChange={(e) => updateValue(index, e.target.value)}
                />
              )}
              {!readonly && (
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => handleRemove(index)}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              )}
            </div>
            {combinedErrors.length > 0 && (
              <ul className="text-xs text-error list-disc list-outside pl-4">
                {combinedErrors.map((err) => (
                  <li key={err}>{err}</li>
                ))}
              </ul>
            )}
          </div>
        );
      })}
      {!readonly && (
        <Button
          variant="outline"
          size="sm"
          className="self-start"
          disabled={!kvList.length ? false : hasErrors}
          onClick={handleAdd}
        >
          <Plus className="w-4 h-4 mr-1" />
          {t("label.add-label")}
        </Button>
      )}
    </div>
  );
}
