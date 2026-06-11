import { useTranslation } from "react-i18next";
import { useRouteError } from "react-router-dom";
import { Button } from "@/react/components/ui/button";

/**
 * Root-route `errorElement`: any uncaught render/loader exception in the
 * route tree lands here instead of react-router's developer-facing default
 * error screen. Shows a recoverable page with the error surfaced so users
 * can copy it into a bug report.
 */
export function RouteErrorPage() {
  const { t } = useTranslation();
  const error = useRouteError();
  const message = error instanceof Error ? error.message : String(error);
  const stack = error instanceof Error ? error.stack : undefined;

  return (
    <div className="flex h-screen w-full flex-col items-center justify-center gap-y-4 px-4">
      <h1 className="text-2xl font-semibold text-main">
        {t("error-page.something-went-wrong")}
      </h1>
      <p className="text-sm text-control-light">
        {t("error-page.unexpected-error-description")}
      </p>
      <div className="flex gap-x-2">
        <Button onClick={() => window.location.reload()}>
          {t("common.refresh")}
        </Button>
        <Button variant="outline" onClick={() => window.location.assign("/")}>
          {t("error-page.go-back-home")}
        </Button>
      </div>
      <details className="w-full max-w-2xl text-sm text-control-light">
        <summary className="cursor-pointer select-none">
          {t("error-page.error-details")}
        </summary>
        <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap rounded-sm border border-control-border bg-control-bg p-3 text-xs">
          {stack ?? message}
        </pre>
      </details>
    </div>
  );
}
