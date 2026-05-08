import { useTranslation } from "react-i18next";
import { cn } from "@/react/lib/utils";

interface EmptyViewProps {
  dark: boolean;
}

export function EmptyView({ dark }: EmptyViewProps) {
  const { t } = useTranslation();
  return (
    <div
      className={cn(
        "text-md font-normal",
        dark ? "text-matrix-green-hover" : "text-control-light"
      )}
    >
      {t("sql-editor.no-data-available")}
    </div>
  );
}
