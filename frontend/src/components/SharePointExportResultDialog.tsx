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

// The Export API reports an upload it could not confirm as UNAVAILABLE.
export function toFailedSharePointExport(
  database: string,
  error: unknown
): SharePointExportResult {
  if (error instanceof ConnectError) {
    return {
      database,
      status: error.code === Code.Unavailable ? "UNCONFIRMED" : "FAILED",
      reason: error.rawMessage,
    };
  }
  return { database, status: "FAILED", reason: String(error) };
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
                {result.status === "EXPORTED" ? (
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
                ) : (
                  <span className="break-words text-control-light">
                    {result.reason}
                  </span>
                )}
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
    case "FAILED":
      return (
        <Badge variant="destructive">
          {t("export-data.sharepoint.status.failed")}
        </Badge>
      );
  }
}
