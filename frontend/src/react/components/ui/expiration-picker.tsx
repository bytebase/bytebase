import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";

/**
 * ExpirationPicker — a datetime picker with inline clear button.
 *
 * Value format: "YYYY-MM-DDTHH:mm" (datetime-local compatible) or undefined
 * for "never expires".
 */
export function ExpirationPicker({
  value,
  onChange,
  minDate,
}: {
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  minDate?: string;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-x-2">
      <Input
        type="datetime-local"
        className="w-64"
        value={value ?? ""}
        min={minDate}
        onChange={(e) => onChange(e.target.value || undefined)}
      />
      {value && (
        <button
          type="button"
          className="p-1 rounded-full hover:bg-gray-200 text-control-placeholder shrink-0"
          onClick={() => onChange(undefined)}
          title={t("common.clear")}
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}
