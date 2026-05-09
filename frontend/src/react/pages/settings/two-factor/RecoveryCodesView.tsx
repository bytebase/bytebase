import { Download } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";

interface RecoveryCodesViewProps {
  recoveryCodes: string[];
  onDownload?: () => void;
}

export function RecoveryCodesView({
  recoveryCodes,
  onDownload,
}: RecoveryCodesViewProps) {
  const { t } = useTranslation();

  const downloadRecoveryCodes = useCallback(() => {
    const content = recoveryCodes.join("\n");
    const blob = new Blob([content], { type: "text/plain" });
    const downloadLink = document.createElement("a");
    downloadLink.href = URL.createObjectURL(blob);
    downloadLink.download = "bytebase-recovery-codes.txt";
    document.body.appendChild(downloadLink);
    downloadLink.click();
    URL.revokeObjectURL(downloadLink.href);
    document.body.removeChild(downloadLink);
    onDownload?.();
  }, [recoveryCodes, onDownload]);

  return (
    <div className="w-full flex flex-col gap-y-4 my-8">
      <Alert
        variant="info"
        title={t(
          "two-factor.setup-steps.download-recovery-codes.keep-safe.self"
        )}
        description={
          <>
            <p>{t("two-factor.setup-steps.download-recovery-codes.tips")}</p>
            <p>
              {t(
                "two-factor.setup-steps.download-recovery-codes.keep-safe.description"
              )}
            </p>
          </>
        }
      />
      <div className="w-full mx-auto flex flex-col justify-start items-start">
        <ul className="w-full grid grid-cols-2 list-disc list-inside mx-auto gap-4 gap-x-24 p-8 px-12 border rounded-xs bg-gray-50">
          {recoveryCodes.map((code) => (
            <li key={code}>
              <code className="ml-2">{code}</code>
            </li>
          ))}
        </ul>
      </div>
      <div className="w-full mx-auto flex flex-row justify-end items-center">
        <Button onClick={downloadRecoveryCodes}>
          <Download className="w-5 h-auto" />
          {t("common.download")}
        </Button>
      </div>
    </div>
  );
}
