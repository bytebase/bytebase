import { useTranslation } from "react-i18next";

export function EmptyView() {
  const { t } = useTranslation();
  return (
    <div className="text-md font-normal text-control-light">
      {t("sql-editor.no-data-available")}
    </div>
  );
}
