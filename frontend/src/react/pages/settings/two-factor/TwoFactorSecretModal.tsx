import { Copy } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { pushNotification } from "@/store";

interface TwoFactorSecretModalProps {
  secret: string;
  open: boolean;
  onClose: () => void;
}

export function TwoFactorSecretModal({
  secret,
  open,
  onClose,
}: TwoFactorSecretModalProps) {
  const { t } = useTranslation();

  const copySecret = async () => {
    try {
      await navigator.clipboard.writeText(secret);
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("two-factor.your-two-factor-secret.copy-succeed"),
      });
    } catch {
      // Clipboard API not available
    }
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="p-6">
        <DialogTitle>{t("two-factor.your-two-factor-secret.self")}</DialogTitle>
        <DialogDescription>
          {t("two-factor.your-two-factor-secret.description")}
        </DialogDescription>
        <div className="my-4 py-2">
          <code className="pr-4">{secret}</code>
        </div>
        <div className="flex items-center justify-end gap-x-2">
          <Button variant="outline" onClick={onClose}>
            {t("common.close")}
          </Button>
          <Button onClick={copySecret}>
            <Copy className="w-4 h-4" />
            {t("common.copy")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
