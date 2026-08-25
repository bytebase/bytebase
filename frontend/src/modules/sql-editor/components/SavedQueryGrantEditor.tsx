import { create as createProto } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import { Trash2, Users } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AccountMultiSelect } from "@/components/AccountMultiSelect";
import { UserCell } from "@/components/UserCell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { getMemberBinding } from "@/lib/memberBindings";
import { extractGroupEmail } from "@/stores";
import { useAppStore } from "@/stores/app";
import { extractUserEmail } from "@/stores/modules/v1/common";
import {
  getUserEmailInBinding,
  groupBindingPrefix,
  userBindingPrefix,
} from "@/types";
import type {
  SavedQuery,
  SavedQueryPolicy,
} from "@/types/proto-es/v1/saved_query_service_pb";
import {
  SavedQueryBinding_Level,
  SavedQueryBindingSchema,
  SavedQueryPolicySchema,
} from "@/types/proto-es/v1/saved_query_service_pb";

type Props = {
  readonly savedQuery: SavedQuery;
  /**
   * Whether the caller may change the grants. A grantee can read the policy —
   * that is how they learn their own level — but only the creator and admins
   * can rewrite it, so the editor renders read-only for everyone else.
   */
  readonly canManage: boolean;
};

/** One grantee and the single level they hold. */
type Grant = {
  readonly member: string;
  readonly level: SavedQueryBinding_Level;
};

const GRANTABLE_LEVELS = [
  SavedQueryBinding_Level.VIEWER,
  SavedQueryBinding_Level.EDITOR,
] as const;

/**
 * Only `user:` and `group:` can be grantees. A service account can create,
 * own, and run its own saved queries, but never receives a grant, so anything
 * else the account picker can produce is rejected rather than silently dropped.
 */
const isGrantableMember = (member: string) =>
  member.startsWith("user:") || member.startsWith("group:");

/**
 * Reads a saved query's sharing policy and, for its creator or an admin,
 * edits it. Adds are staged: picked accounts sit as chips until the Add
 * button commits them at the invite level in one write (the BigQuery /
 * Snowsight / Databricks confirm-the-add pattern), while each grantee row's
 * level select and remove button apply per action. Every write rewrites the
 * policy in full under the etag the read returned; a concurrent change
 * aborts the write so a revocation someone else just made is never silently
 * reinstated.
 */
export function SavedQueryGrantEditor({ savedQuery, canManage }: Props) {
  const { t } = useTranslation();
  const getSavedQueryPolicy = useAppStore((s) => s.getSavedQueryPolicy);
  const setSavedQueryPolicy = useAppStore((s) => s.setSavedQueryPolicy);
  const batchGetOrFetchUsers = useAppStore((s) => s.batchGetOrFetchUsers);
  const batchGetOrFetchGroups = useAppStore((s) => s.batchGetOrFetchGroups);
  const notify = useAppStore((s) => s.notify);
  const currentUserEmail = useAppStore((s) => s.currentUser?.email);

  const [policy, setPolicy] = useState<SavedQueryPolicy | undefined>();
  const [inviteLevel, setInviteLevel] = useState<SavedQueryBinding_Level>(
    SavedQueryBinding_Level.VIEWER
  );
  // Accounts picked but not yet granted: chips in the picker until Add.
  const [pending, setPending] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  // Focus recovery target after a commit/remove unmounts the focused control.
  const sectionRef = useRef<HTMLElement>(null);
  // Rejects policy responses that land after a newer load started (the
  // popover instance can outlive a switch to a different saved query).
  const loadGeneration = useRef(0);

  const load = useCallback(async () => {
    const generation = ++loadGeneration.current;
    setLoading(true);
    try {
      const fetched = await getSavedQueryPolicy(savedQuery.name);
      if (generation !== loadGeneration.current) return;
      setPolicy(fetched);
      // Prefetch grantee display names; rows fall back to emails if this fails.
      const members = fetched.bindings.flatMap((binding) => binding.members);
      void Promise.allSettled([
        batchGetOrFetchUsers(
          members.filter((member) => !member.startsWith(groupBindingPrefix))
        ),
        batchGetOrFetchGroups(
          members.filter((member) => member.startsWith(groupBindingPrefix))
        ),
      ]);
    } catch {
      if (generation !== loadGeneration.current) return;
      setPolicy(undefined);
    } finally {
      if (generation === loadGeneration.current) setLoading(false);
    }
  }, [
    batchGetOrFetchGroups,
    batchGetOrFetchUsers,
    getSavedQueryPolicy,
    savedQuery.name,
  ]);

  useEffect(() => {
    // `load` changes identity exactly when `savedQuery.name` does; a
    // different saved query is a fresh editor, so nothing staged — and no
    // previously loaded policy or its etag — may carry over.
    setPolicy(undefined);
    setPending([]);
    setInviteLevel(SavedQueryBinding_Level.VIEWER);
    setSaving(false);
    void load();
  }, [load]);

  // One row per grantee, grouped viewers first; within a level, rows keep the
  // policy's member order. A level change moves its row to the end of the new
  // level's group (writes append the touched member). A member duplicated
  // across levels (which the server rejects) collapses to their first listing.
  const grants = useMemo<Grant[]>(() => {
    if (!policy) return [];
    const rows: Grant[] = [];
    const seen = new Set<string>();
    for (const level of GRANTABLE_LEVELS) {
      for (const binding of policy.bindings) {
        if (binding.level !== level) continue;
        for (const member of binding.members) {
          if (seen.has(member)) continue;
          seen.add(member);
          rows.push({ member, level });
        }
      }
    }
    return rows;
  }, [policy]);

  // The creator renders as a pinned Owner row, not a grant row: ownership is
  // not a binding. A degenerate policy that does list the creator keeps its
  // binding through rewrites — it is only hidden from the rows.
  const creatorMember = useMemo(() => {
    const email = extractUserEmail(savedQuery.creator);
    return email && email !== savedQuery.creator
      ? getUserEmailInBinding(email)
      : "";
  }, [savedQuery.creator]);

  const grantRows = useMemo(
    () => grants.filter((grant) => grant.member !== creatorMember),
    [grants, creatorMember]
  );

  // Whether the write succeeded — Add keeps its staged chips on failure.
  const writeGrants = async (next: Grant[]): Promise<boolean> => {
    if (!policy) return false;
    // The write belongs to the editor state it started from; once a switch to
    // another saved query bumps the generation, its completion must neither
    // replace the new policy nor reload/toast for the old query.
    const generation = loadGeneration.current;
    const bindings = [
      ...GRANTABLE_LEVELS.map((level) =>
        createProto(SavedQueryBindingSchema, {
          level,
          members: next
            .filter((grant) => grant.level === level)
            .map((grant) => grant.member),
        })
      ).filter((binding) => binding.members.length > 0),
      // Levels this bundle does not know (a newer server's) pass through
      // untouched — the same carry-through the creator binding gets.
      ...policy.bindings.filter(
        (binding) =>
          !(GRANTABLE_LEVELS as readonly SavedQueryBinding_Level[]).includes(
            binding.level
          )
      ),
    ];

    setSaving(true);
    try {
      const updated = await setSavedQueryPolicy(
        savedQuery.name,
        createProto(SavedQueryPolicySchema, { bindings, etag: policy.etag })
      );
      if (generation !== loadGeneration.current) return false;
      setPolicy(updated);
      return true;
    } catch (error) {
      if (generation !== loadGeneration.current) return false;
      if (error instanceof ConnectError && error.code === Code.Aborted) {
        // Somebody else changed the grants between the read and this write.
        // Reload rather than retrying, so their change is not overwritten.
        notify({
          module: "bytebase",
          style: "WARN",
          title: t("sql-editor.saved-query-share.policy-changed"),
        });
        // The write is over; clear saving before the reload, whose own
        // generation bump makes the guarded finally skip it.
        setSaving(false);
        await load();
        return false;
      }
      notify({
        module: "bytebase",
        style: "CRITICAL",
        title:
          error instanceof ConnectError ? error.message : t("common.error"),
      });
      return false;
    } finally {
      if (generation === loadGeneration.current) setSaving(false);
    }
  };

  // Hidden from the picker: accounts a grant would be pointless or redundant
  // for. The creator and the caller hold access already; grantees change
  // level through their row, not by re-adding.
  const excludeAccounts = useMemo(() => {
    const excluded = new Set(grants.map((grant) => grant.member));
    if (currentUserEmail) excluded.add(getUserEmailInBinding(currentUserEmail));
    if (creatorMember) excluded.add(creatorMember);
    return [...excluded];
  }, [grants, currentUserEmail, creatorMember]);

  const stagePending = (members: string[]) => {
    const rejected = members.filter((member) => !isGrantableMember(member));
    if (rejected.length > 0) {
      notify({
        module: "bytebase",
        style: "WARN",
        title: t("sql-editor.saved-query-share.only-users-and-groups"),
      });
    }
    const granted = members.filter(isGrantableMember);
    if (granted.length === 0) {
      // Empty staging ends the compose session: the next one starts back at
      // the safe default, matching a committed Add.
      setInviteLevel(SavedQueryBinding_Level.VIEWER);
    }
    setPending(granted);
  };

  const handleAdd = async () => {
    if (pending.length === 0) return;
    // A member holds one level at a time: adding an existing grantee (still
    // possible through a concurrent grant elsewhere) moves them to the invite
    // level instead of granting both.
    const touched = new Set(pending);
    const next = [
      ...grants.filter((grant) => !touched.has(grant.member)),
      ...pending.map((member) => ({ member, level: inviteLevel })),
    ];
    if (sameGrants(grants, next)) {
      setPending([]);
      setInviteLevel(SavedQueryBinding_Level.VIEWER);
      return;
    }
    if (await writeGrants(next)) {
      setPending([]);
      // Each add is its own compose session; the next one starts back at the
      // safe default.
      setInviteLevel(SavedQueryBinding_Level.VIEWER);
      // The commit row just unmounted under the focused Add button.
      sectionRef.current?.focus();
    }
  };

  const changeLevel = (member: string, level: SavedQueryBinding_Level) => {
    const current = grants.find((grant) => grant.member === member);
    if (!current || current.level === level) return;
    void writeGrants([
      ...grants.filter((grant) => grant.member !== member),
      { member, level },
    ]);
  };

  const removeMember = async (member: string) => {
    if (await writeGrants(grants.filter((grant) => grant.member !== member))) {
      // The row holding the focused remove button just unmounted.
      sectionRef.current?.focus();
    }
  };

  if (loading) {
    return (
      <p className="text-sm text-control-light">{t("common.loading")}...</p>
    );
  }
  if (!policy) {
    // A failed read must not look like "nothing to manage" — say so and
    // offer a retry in place.
    return (
      <div className="flex items-center gap-x-2 text-sm text-control-light">
        <span>{t("sql-editor.saved-query-share.load-failed")}</span>
        <Button
          type="button"
          appearance="link"
          size="sm"
          onClick={() => void load()}
        >
          {t("common.refresh")}
        </Button>
      </div>
    );
  }

  return (
    <section
      ref={sectionRef}
      tabIndex={-1}
      className="flex flex-col gap-y-2 focus:outline-hidden"
    >
      {canManage && (
        <div data-testid="grant-invite" className="flex flex-col gap-y-2">
          <AccountMultiSelect
            value={pending}
            onChange={stagePending}
            disabled={saving}
            excludeAccounts={excludeAccounts}
            placeholder={t("sql-editor.saved-query-share.add-people")}
          />
          {/* The commit controls exist only while an add is in progress —
              the resting popover is just the field and the list. */}
          {pending.length > 0 && (
            <div className="flex justify-end gap-x-2">
              <LevelSelect
                value={inviteLevel}
                onChange={setInviteLevel}
                disabled={saving}
                className="min-w-24"
              />
              <Button
                type="button"
                data-testid="grant-add"
                disabled={saving}
                onClick={() => void handleAdd()}
              >
                {t("common.add")}
              </Button>
            </div>
          )}
        </div>
      )}

      {(creatorMember || grantRows.length > 0) && (
        <>
          {/* Caption, not a heading: the list states who holds access; the
              field above is where the action lives. */}
          <p
            id="saved-query-grantee-caption"
            className="text-xs text-control-light"
          >
            {t("sql-editor.saved-query-share.people-with-access")}
          </p>
          <ul
            aria-labelledby="saved-query-grantee-caption"
            className="flex flex-col gap-y-1 max-h-64 overflow-y-auto"
          >
            {creatorMember && (
              <li
                data-testid="grant-owner-row"
                className="flex items-center justify-between gap-x-2"
              >
                <GranteeCell
                  member={creatorMember}
                  badge={
                    currentUserEmail &&
                    creatorMember ===
                      getUserEmailInBinding(currentUserEmail) ? (
                      <Badge className="text-xs">{t("common.you")}</Badge>
                    ) : undefined
                  }
                />
                <span className="text-sm text-control-light shrink-0">
                  {t("sql-editor.saved-query-share.owner")}
                </span>
              </li>
            )}
            {grantRows.map(({ member, level }) => (
              <li
                key={member}
                data-testid="grant-row"
                data-member={member}
                className="flex items-center justify-between gap-x-2"
              >
                <GranteeCell member={member} />
                {canManage ? (
                  <div className="flex items-center gap-x-2 shrink-0">
                    <LevelSelect
                      value={level}
                      onChange={(next) => changeLevel(member, next)}
                      disabled={saving}
                      size="sm"
                      className="min-w-24"
                    />
                    <Button
                      type="button"
                      appearance="secondary"
                      size="sm"
                      aria-label={t("common.remove")}
                      disabled={saving}
                      className="text-control-light hover:text-error"
                      onClick={() => void removeMember(member)}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                ) : (
                  <span className="text-sm text-control-light shrink-0">
                    {levelLabel(t, level)}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </>
      )}

      {grantRows.length === 0 &&
        (canManage ? (
          <p className="text-xs text-control-light">
            {t("sql-editor.saved-query-share.private-hint")}
          </p>
        ) : (
          // With an Owner row on screen, "not shared" would contradict it;
          // the single-row list already says everything.
          !creatorMember && (
            <p className="text-xs text-control-light">
              {t("sql-editor.saved-query-share.not-shared")}
            </p>
          )
        ))}
    </section>
  );
}

/** Same member→level assignments, ignoring row order. */
const sameGrants = (a: Grant[], b: Grant[]) => {
  if (a.length !== b.length) return false;
  const levels = new Map(a.map((grant) => [grant.member, grant.level]));
  return b.every((grant) => levels.get(grant.member) === grant.level);
};

function LevelSelect({
  value,
  onChange,
  disabled,
  size = "md",
  className,
}: {
  readonly value: SavedQueryBinding_Level;
  readonly onChange: (level: SavedQueryBinding_Level) => void;
  readonly disabled?: boolean;
  readonly size?: "sm" | "md";
  readonly className?: string;
}) {
  const { t } = useTranslation();
  return (
    <Select
      value={value}
      onValueChange={(next: SavedQueryBinding_Level | null) => {
        if (next !== null) onChange(next);
      }}
      disabled={disabled}
    >
      <SelectTrigger size={size} className={className}>
        <SelectValue>
          {(current: SavedQueryBinding_Level) => levelLabel(t, current)}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {GRANTABLE_LEVELS.map((candidate) => (
          <SelectItem key={candidate} value={candidate}>
            {levelLabel(t, candidate)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/**
 * Level → label, keeping the locale keys as literal arguments so the locale
 * checker can trace them statically.
 */
const levelLabel = (t: TFunction, level: SavedQueryBinding_Level) =>
  level === SavedQueryBinding_Level.EDITOR
    ? t("sql-editor.saved-query-share.editor")
    : t("sql-editor.saved-query-share.viewer");

function GranteeCell({
  member,
  badge,
}: {
  readonly member: string;
  readonly badge?: ReactNode;
}) {
  const isGroup = member.startsWith(groupBindingPrefix);
  // Subscribe to the store slice behind this row so it repaints when the
  // prefetch lands; the display projection itself is the shared members
  // surface, which handles every principal kind.
  useAppStore((s) =>
    isGroup ? s.getGroupByIdentifier(member) : s.getUserByIdentifier(member)
  );
  const memberBinding = getMemberBinding(member, "");
  if (!memberBinding) return null;

  const email = memberBinding.group
    ? extractGroupEmail(memberBinding.group.name)
    : (memberBinding.user?.email ?? "");
  const title = memberBinding.title || email;
  return (
    <UserCell
      size="sm"
      className="min-w-0"
      title={title}
      subtitle={email && title !== email ? email : undefined}
      avatar={
        isGroup ? (
          <div className="size-7 rounded-full bg-control-bg text-control-light flex items-center justify-center shrink-0">
            <Users className="size-4" />
          </div>
        ) : undefined
      }
      hoverEmail={member.startsWith(userBindingPrefix) ? email : undefined}
      badges={badge}
    />
  );
}
