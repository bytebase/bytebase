import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import { INLINE_EDIT_INPUT_CLASS } from "../common";

interface DefaultValue {
  hasDefault: boolean;
  default: string;
}

interface Props {
  column: ColumnMetadata;
  disabled: boolean;
  onUpdate: (value: DefaultValue) => void;
}

export function DefaultValueCell({ column, disabled, onUpdate }: Props) {
  const { t } = useTranslation();

  const handleChange = useCallback(
    (value: string) => {
      onUpdate({
        hasDefault: !!value.trim(),
        default: value.trim(),
      });
    },
    [onUpdate]
  );

  return (
    <Input
      value={column.default ?? ""}
      disabled={disabled}
      placeholder={t("schema-editor.default.placeholder")}
      size="xs"
      className={INLINE_EDIT_INPUT_CLASS}
      onChange={(e) => handleChange(e.target.value)}
    />
  );
}
