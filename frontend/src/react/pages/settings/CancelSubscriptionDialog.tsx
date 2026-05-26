import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { Textarea } from "@/react/components/ui/textarea";

// Stripe cancellation_details.feedback enum values.
// See https://stripe.com/docs/api/subscriptions/cancel
const CANCEL_REASONS = [
  { value: "too_expensive", labelKey: "too-expensive" },
  { value: "missing_features", labelKey: "missing-features" },
  { value: "unused", labelKey: "unused" },
  { value: "switched_service", labelKey: "switched-service" },
  { value: "too_complex", labelKey: "too-complex" },
  { value: "low_quality", labelKey: "low-quality" },
  { value: "customer_service", labelKey: "customer-service" },
  { value: "other", labelKey: "other" },
] as const;

const COMMENT_MAX_LENGTH = 500;

interface CancelSubscriptionDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly onConfirm: (feedback: string, comment: string) => Promise<void>;
}

export function CancelSubscriptionDialog({
  open,
  onOpenChange,
  onConfirm,
}: CancelSubscriptionDialogProps) {
  const { t } = useTranslation();
  const [feedback, setFeedback] = useState<string>("");
  const [comment, setComment] = useState<string>("");
  const [submitting, setSubmitting] = useState(false);

  // Reset state every time the dialog opens.
  useEffect(() => {
    if (open) {
      setFeedback("");
      setComment("");
      setSubmitting(false);
    }
  }, [open]);

  const handleConfirm = async () => {
    if (!feedback || submitting) return;
    setSubmitting(true);
    try {
      await onConfirm(feedback, comment.trim());
      onOpenChange(false);
    } catch {
      setSubmitting(false);
    }
  };

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (submitting) return;
        onOpenChange(next);
      }}
    >
      <AlertDialogContent className="max-w-lg">
        <AlertDialogTitle>
          {t("subscription.purchase.cancel-dialog.title")}
        </AlertDialogTitle>
        <AlertDialogDescription>
          {t("subscription.purchase.cancel-dialog.description")}
        </AlertDialogDescription>

        <div className="mt-4 flex flex-col gap-y-4">
          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-main">
              {t("subscription.purchase.cancel-dialog.reason-label")}
            </div>
            <RadioGroup
              value={feedback}
              onValueChange={(value) =>
                setFeedback(typeof value === "string" ? value : "")
              }
              className="flex flex-col items-start gap-y-2"
            >
              {CANCEL_REASONS.map((reason) => (
                <RadioGroupItem key={reason.value} value={reason.value}>
                  {t(
                    `subscription.purchase.cancel-dialog.reason.${reason.labelKey}`
                  )}
                </RadioGroupItem>
              ))}
            </RadioGroup>
          </div>

          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-main">
              {t("subscription.purchase.cancel-dialog.comment-label")}
            </div>
            <Textarea
              value={comment}
              maxLength={COMMENT_MAX_LENGTH}
              rows={3}
              placeholder={t(
                "subscription.purchase.cancel-dialog.comment-placeholder"
              )}
              onChange={(e) => setComment(e.target.value)}
            />
            <div className="text-xs text-control-placeholder text-right">
              {comment.length}/{COMMENT_MAX_LENGTH}
            </div>
          </div>
        </div>

        <AlertDialogFooter>
          <Button
            variant="outline"
            disabled={submitting}
            onClick={() => onOpenChange(false)}
          >
            {t("subscription.purchase.cancel-dialog.keep")}
          </Button>
          <Button
            variant="destructive"
            disabled={!feedback || submitting}
            onClick={handleConfirm}
          >
            {submitting && <Loader2 className="mr-2 size-4 animate-spin" />}
            {t("subscription.purchase.cancel-dialog.confirm")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
