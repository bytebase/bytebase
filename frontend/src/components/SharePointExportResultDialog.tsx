import { Code, ConnectError } from "@connectrpc/connect";
import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

// SharePoint URLs arrive percent-encoded; show them as people typed them.
export function formatSharePointUrl(url: string): string {
  try {
    return decodeURI(url);
  } catch {
    return url;
  }
}

export type SharePointExportResult = {
  database: string;
} & (
  | { status: "EXPORTED"; uri: string }
  | { status: "FAILED" | "UNCONFIRMED"; reason: string }
  // CANCELED: canceled while this database was exporting, so files may exist.
  // NOT_STARTED: canceled before this database's export was requested.
  | { status: "CANCELED" | "NOT_STARTED" }
);

// `uri` is the folder created for this export; falls back to the destination
// URL when the response omits it.
export function toExportedSharePointResult(
  database: string,
  uri: string,
  destinationUrl: string
): SharePointExportResult {
  return { database, status: "EXPORTED", uri: uri || destinationUrl };
}

// The Export API reports an upload it could not confirm as UNAVAILABLE. An
// aborted request rejects with CANCELED.
export function toFailedSharePointExport(
  database: string,
  error: unknown
): SharePointExportResult {
  if (error instanceof ConnectError) {
    if (error.code === Code.Canceled) {
      return { database, status: "CANCELED" };
    }
    return {
      database,
      status: error.code === Code.Unavailable ? "UNCONFIRMED" : "FAILED",
      reason: error.rawMessage,
    };
  }
  return { database, status: "FAILED", reason: String(error) };
}

export function toNotStartedSharePointExport(
  database: string
): SharePointExportResult {
  return { database, status: "NOT_STARTED" };
}

interface Props {
  results: SharePointExportResult[];
  onClose: () => void;
}

export function SharePointExportResultDialog({ results, onClose }: Props) {
  const { t } = useTranslation();
  const allExported = results.every((r) => r.status === "EXPORTED");

  return (
    <Dialog
      open={results.length > 0}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="max-w-lg">
        <div className="flex flex-col gap-y-4">
          <DialogTitle>
            {allExported
              ? t("export-data.sharepoint.exported-title")
              : t("export-data.sharepoint.result-title")}
          </DialogTitle>
          <ul className="flex flex-col divide-y divide-block-border rounded-sm border border-block-border">
            {results.map((result) => (
              <li
                key={result.database}
                className="flex flex-col gap-y-1 px-4 py-3 text-sm"
              >
                <div className="flex items-center justify-between gap-x-2">
                  <span className="truncate font-medium">
                    {result.database}
                  </span>
                  <ExportStatusBadge status={result.status} />
                </div>
                <ExportResultDetail result={result} />
              </li>
            ))}
          </ul>
          <div className="flex justify-end">
            <Button onClick={onClose}>{t("common.close")}</Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ExportResultDetail({ result }: { result: SharePointExportResult }) {
  const { t } = useTranslation();
  switch (result.status) {
    case "EXPORTED":
      return (
        result.uri && (
          <>
            <span className="break-all text-control-light">
              {formatSharePointUrl(result.uri)}
            </span>
            <a
              href={result.uri}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex w-fit items-center gap-x-1 text-accent hover:underline"
            >
              {t("export-data.sharepoint.open")}
              <ExternalLink className="size-3" />
            </a>
          </>
        )
      );
    case "CANCELED":
      return (
        <span className="break-words text-control-light">
          {t("export-data.sharepoint.canceled-description")}
        </span>
      );
    case "NOT_STARTED":
      return (
        <span className="break-words text-control-light">
          {t("export-data.sharepoint.not-started-description")}
        </span>
      );
    default:
      return (
        <span className="break-words text-control-light">{result.reason}</span>
      );
  }
}

function ExportStatusBadge({
  status,
}: {
  status: SharePointExportResult["status"];
}) {
  const { t } = useTranslation();
  switch (status) {
    case "EXPORTED":
      return (
        <Badge variant="success">
          {t("export-data.sharepoint.status.exported")}
        </Badge>
      );
    case "UNCONFIRMED":
      return (
        <Badge variant="warning">
          {t("export-data.sharepoint.status.unconfirmed")}
        </Badge>
      );
    case "CANCELED":
      return (
        <Badge variant="warning">
          {t("export-data.sharepoint.status.canceled")}
        </Badge>
      );
    case "NOT_STARTED":
      return (
        <Badge variant="default">
          {t("export-data.sharepoint.status.not-started")}
        </Badge>
      );
    case "FAILED":
      return (
        <Badge variant="destructive">
          {t("export-data.sharepoint.status.failed")}
        </Badge>
      );
  }
}
