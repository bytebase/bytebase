import dayjs from "dayjs";
import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { DateTimePicker } from "@/react/components/ui/date-time-picker";

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
  maxDate,
  className,
}: {
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  minDate?: string;
  maxDate?: string;
  className?: string;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-x-2">
      <DateTimePicker
        value={value ? new Date(value) : undefined}
        minDate={minDate ? new Date(minDate) : undefined}
        maxDate={maxDate ? new Date(maxDate) : undefined}
        onChange={(date) =>
          onChange(date ? dayjs(date).format("YYYY-MM-DDTHH:mm") : undefined)
        }
        className={className}
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
