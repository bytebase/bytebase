import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { UserAvatar } from "@/components/UserAvatar";
import { Badge } from "@/components/ui/badge";
import {
  PreviewCard,
  PreviewCardContent,
  PreviewCardTrigger,
} from "@/components/ui/preview-card";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";

interface UserHoverCardProps {
  /** Email identifying the user. The card fetches by this on open. */
  email: string;
  /** Name already known to the caller, shown until the fetch resolves. */
  fallbackTitle?: string;
  children: ReactNode;
}

/**
 * Identity at a glance for a name or avatar mentioned in passing.
 *
 * Deliberately limited to who the person is: display name, email, and whether
 * the account is still active. Roles, groups and activity timestamps are not
 * here — they answer a different question, and last-login in particular would
 * expose every member's activity pattern to every other member, since reading
 * a user record needs no special permission. Administering an account happens
 * on the Users page.
 */
export function UserHoverCard({
  email,
  fallbackTitle,
  children,
}: UserHoverCardProps) {
  const { t } = useTranslation();
  const getOrFetchUserByIdentifier = useAppStore(
    (state) => state.getOrFetchUserByIdentifier
  );
  const user = useAppStore((state) => state.getUserByIdentifier(email));

  // Fetch on open rather than on mount: the card hangs off every name in a
  // table, so fetching eagerly would issue one request per row. The store
  // returns a cached user without a request, so re-asking on each open costs
  // nothing and lets a card that failed once recover on the next hover.
  const handleOpenChange = (open: boolean) => {
    if (!open) return;
    void getOrFetchUserByIdentifier({ identifier: email, silent: true });
  };

  const title = user?.title || fallbackTitle || email;
  const isDeleted = user?.state === State.DELETED;

  return (
    <PreviewCard onOpenChange={handleOpenChange}>
      <PreviewCardTrigger
        render={
          <span
            className="inline-flex items-center rounded-xs focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            tabIndex={0}
          />
        }
      >
        {children}
      </PreviewCardTrigger>
      {/* The card portals into the overlay root, but React events still
          propagate along the component tree — without this, selecting the
          email inside the card would trigger the enclosing row's click. */}
      <PreviewCardContent
        className="max-w-72"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start gap-x-3">
          <UserAvatar title={title} colorSeed={email} />
          <div className="flex min-w-0 flex-col gap-y-0.5">
            <div className="flex items-center gap-x-1.5">
              <span className="truncate font-medium text-main">{title}</span>
              {isDeleted && (
                <Badge className="shrink-0 text-xs px-1.5 py-0">
                  {t("common.deactivated")}
                </Badge>
              )}
            </div>
            <span className="truncate text-xs text-control-light">{email}</span>
          </div>
        </div>
      </PreviewCardContent>
    </PreviewCard>
  );
}
