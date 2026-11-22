import { LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

export function ExportGrantLookupStatus({
  loading,
  failed,
  onRetry,
}: {
  loading: boolean;
  failed: boolean;
  onRetry: () => void;
}) {
  const { t } = useTranslation();
  if (loading)
    return (
      <Button size="sm" appearance="secondary" disabled>
        <LoaderCircle className="animate-spin" />
        {t("sql-editor.export-grant-lookup-loading")}
      </Button>
    );
  if (!failed) return null;
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Alert variant="error" appearance="caption" role="alert">
        {t("sql-editor.export-grant-lookup-failed")}
      </Alert>
      <Button size="sm" appearance="secondary" onClick={onRetry}>
        {t("sql-editor.export-grant-lookup-retry")}
      </Button>
    </div>
  );
}
