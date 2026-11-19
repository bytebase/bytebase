import {
  Ban,
  CircleCheck,
  Ellipsis,
  ExternalLink,
  Loader2,
} from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent } from "@/components/ui/popover";
import { Tooltip } from "@/components/ui/tooltip";
import { useTimeReading } from "@/hooks/useTimeReading";
import {
  type AccessGrantDisplayStatus,
  accessGrantExpirationReading,
  isAccessGrantExpired,
} from "@/utils/accessGrant";

type GrantAction = "activate" | "revoke";

export function AccessGrantActions({
  issue,
  status,
  expireTimeMs,
  canActivate,
  canRevoke,
  onActivate,
  onRevoke,
}: {
  issue?: string;
  status: AccessGrantDisplayStatus;
  expireTimeMs?: number;
  canActivate: boolean;
  canRevoke: boolean;
  onActivate: () => void | Promise<void>;
  onRevoke: () => void | Promise<void>;
}) {
  const { t } = useTranslation();
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmAction, setConfirmAction] = useState<GrantAction | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const skipMenuFinalFocusRef = useRef(false);
  const submittingRef = useRef(false);
  const titleId = useId();
  const descriptionId = useId();
  const expired = useTimeReading(accessGrantExpirationReading, expireTimeMs);
  const action: GrantAction | null = expired
    ? null
    : status === "ACTIVE" && canRevoke
      ? "revoke"
      : status === "REVOKED" && canActivate
        ? "activate"
        : null;
  const label =
    action === "revoke"
      ? t("sql-editor.revoke-access")
      : t("sql-editor.activate-access");
  const ActionIcon = action === "revoke" ? Ban : CircleCheck;

  useEffect(() => {
    setMenuOpen(false);
    setConfirmAction(null);
    setError(null);
  }, [action]);

  const handleConfirm = async () => {
    if (!action || confirmAction !== action || submittingRef.current) return;
    if (expireTimeMs !== undefined && isAccessGrantExpired(expireTimeMs)) {
      setConfirmAction(null);
      return;
    }
    submittingRef.current = true;
    setIsSubmitting(true);
    setError(null);
    try {
      await (action === "activate" ? onActivate() : onRevoke());
      setConfirmAction(null);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      submittingRef.current = false;
      setIsSubmitting(false);
    }
  };

  return (
    <div className="flex items-center justify-end gap-2">
      {issue ? (
        <Tooltip content={t("sql-editor.view-issue")}>
          <RouterLink
            to={issue.startsWith("/") ? issue : `/${issue}`}
            target="_blank"
            rel="noreferrer"
            aria-label={t("sql-editor.view-issue")}
            className={buttonVariants({
              appearance: "secondary",
              size: "sm",
              className: "shrink-0",
            })}
            onClick={(event) => event.stopPropagation()}
          >
            <ExternalLink className="size-4" aria-hidden="true" />
          </RouterLink>
        </Tooltip>
      ) : (
        <span className="w-8 shrink-0" aria-hidden="true" />
      )}
      {action ? (
        <>
          <DropdownMenu
            open={menuOpen}
            onOpenChange={(open) => {
              if (open) skipMenuFinalFocusRef.current = false;
              setMenuOpen(open);
            }}
          >
            <Tooltip content={t("common.more")}>
              <DropdownMenuTrigger
                ref={triggerRef}
                render={
                  <Button
                    appearance="secondary"
                    size="sm"
                    aria-label={t("common.more")}
                    disabled={isSubmitting}
                    onClick={(event) => event.stopPropagation()}
                  />
                }
              >
                <Ellipsis className="size-4" aria-hidden="true" />
              </DropdownMenuTrigger>
            </Tooltip>
            <DropdownMenuContent
              finalFocus={() =>
                skipMenuFinalFocusRef.current ? false : triggerRef.current
              }
            >
              <DropdownMenuItem
                className={
                  action === "revoke"
                    ? "text-error data-highlighted:bg-error/10"
                    : undefined
                }
                onClick={(event) => {
                  event.stopPropagation();
                  skipMenuFinalFocusRef.current = true;
                  setMenuOpen(false);
                  setError(null);
                  setConfirmAction(action);
                }}
              >
                <ActionIcon className="size-4" aria-hidden="true" />
                {label}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Popover
            open={confirmAction === action}
            onOpenChange={(open) => {
              if (!open && !submittingRef.current) setConfirmAction(null);
            }}
          >
            <PopoverContent
              anchor={triggerRef}
              finalFocus={triggerRef}
              align="end"
              className="w-72 max-w-[var(--available-width)]"
              aria-labelledby={titleId}
              aria-describedby={descriptionId}
              aria-busy={isSubmitting}
              onClick={(event) => event.stopPropagation()}
            >
              <div className="flex flex-col gap-3">
                <p id={titleId} className="font-medium">
                  {label}
                </p>
                <p id={descriptionId} className="text-control-light">
                  {action === "revoke"
                    ? t("sql-editor.revoke-confirm")
                    : t("sql-editor.activate-confirm")}
                </p>
                {error !== null && (
                  <Alert
                    variant="error"
                    appearance="caption"
                    className="break-words"
                  >
                    {error}
                  </Alert>
                )}
                <div className="flex flex-wrap justify-end gap-2">
                  <Button
                    appearance="outline"
                    size="sm"
                    disabled={isSubmitting}
                    onClick={() => setConfirmAction(null)}
                  >
                    {t("common.cancel")}
                  </Button>
                  <Button
                    size="sm"
                    variant={action === "revoke" ? "destructive" : "default"}
                    disabled={isSubmitting}
                    onClick={() => void handleConfirm()}
                  >
                    {isSubmitting && (
                      <Loader2
                        className="size-4 animate-spin"
                        aria-hidden="true"
                      />
                    )}
                    {label}
                  </Button>
                </div>
              </div>
            </PopoverContent>
          </Popover>
        </>
      ) : (
        <span className="w-8 shrink-0" aria-hidden="true" />
      )}
    </div>
  );
}
