import { create as createProto } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AccountMultiSelect } from "@/components/AccountMultiSelect";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAppStore } from "@/stores/app";
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
 * edits it. Writes replace the policy in full under the etag the read
 * returned; a concurrent change aborts the write so a revocation someone else
 * just made is never silently reinstated.
 */
export function SavedQueryGrantEditor({ savedQuery, canManage }: Props) {
  const { t } = useTranslation();
  const getSavedQueryPolicy = useAppStore((s) => s.getSavedQueryPolicy);
  const setSavedQueryPolicy = useAppStore((s) => s.setSavedQueryPolicy);
  const notify = useAppStore((s) => s.notify);

  const [policy, setPolicy] = useState<SavedQueryPolicy | undefined>();
  const [level, setLevel] = useState<SavedQueryBinding_Level>(
    SavedQueryBinding_Level.VIEWER
  );
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setPolicy(await getSavedQueryPolicy(savedQuery.name));
    } catch {
      setPolicy(undefined);
    } finally {
      setLoading(false);
    }
  }, [getSavedQueryPolicy, savedQuery.name]);

  useEffect(() => {
    void load();
  }, [load]);

  // The picker edits one level at a time, so members are flattened into the
  // selected level's list and rewritten as a single binding on save.
  const membersAtLevel = useMemo(
    () =>
      policy?.bindings.find((binding) => binding.level === level)?.members ??
      [],
    [policy, level]
  );

  const membersAtOtherLevel = useMemo(
    () =>
      policy?.bindings
        .filter((binding) => binding.level !== level)
        .flatMap((binding) => binding.members) ?? [],
    [policy, level]
  );

  const handleChange = async (members: string[]) => {
    if (!policy) return;
    const rejected = members.filter((member) => !isGrantableMember(member));
    if (rejected.length > 0) {
      notify({
        module: "bytebase",
        style: "WARN",
        title: t("sql-editor.saved-query-share.only-users-and-groups"),
      });
    }
    const granted = members.filter(isGrantableMember);
    // A member holds one level at a time: promoting or demoting somebody drops
    // them from the level they used to hold rather than leaving both.
    const promoted = new Set(granted);

    const bindings = [];
    for (const candidate of GRANTABLE_LEVELS) {
      const at =
        candidate === level
          ? granted
          : (policy.bindings
              .find((binding) => binding.level === candidate)
              ?.members.filter((member) => !promoted.has(member)) ?? []);
      if (at.length > 0) {
        bindings.push(
          createProto(SavedQueryBindingSchema, {
            level: candidate,
            members: at,
          })
        );
      }
    }

    setSaving(true);
    try {
      const next = await setSavedQueryPolicy(
        savedQuery.name,
        createProto(SavedQueryPolicySchema, {
          bindings,
          etag: policy.etag,
        })
      );
      setPolicy(next);
    } catch (error) {
      if (error instanceof ConnectError && error.code === Code.Aborted) {
        // Somebody else changed the grants between the read and this write.
        // Reload rather than retrying, so their change is not overwritten.
        notify({
          module: "bytebase",
          style: "WARN",
          title: t("sql-editor.saved-query-share.policy-changed"),
        });
        await load();
        return;
      }
      notify({
        module: "bytebase",
        style: "CRITICAL",
        title:
          error instanceof ConnectError ? error.message : t("common.error"),
      });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <p className="text-sm text-control-light">{t("common.loading")}...</p>
    );
  }
  if (!policy) {
    return null;
  }

  return (
    <section className="flex flex-col gap-y-2">
      <div className="flex items-center justify-between gap-x-2">
        <h3 className="text-sm font-medium">
          {t("sql-editor.saved-query-share.shared-with")}
        </h3>
        {canManage && (
          <Select
            value={String(level)}
            onValueChange={(next) =>
              setLevel(Number(next) as SavedQueryBinding_Level)
            }
          >
            <SelectTrigger size="sm" className="w-28">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={String(SavedQueryBinding_Level.VIEWER)}>
                {t("sql-editor.saved-query-share.viewer")}
              </SelectItem>
              <SelectItem value={String(SavedQueryBinding_Level.EDITOR)}>
                {t("sql-editor.saved-query-share.editor")}
              </SelectItem>
            </SelectContent>
          </Select>
        )}
      </div>

      {canManage ? (
        <>
          <AccountMultiSelect
            value={membersAtLevel}
            onChange={(members) => void handleChange(members)}
            disabled={saving}
          />
          {membersAtOtherLevel.length > 0 && (
            <p className="text-xs text-control-light">
              {t("sql-editor.saved-query-share.granted-at-other-level", {
                count: membersAtOtherLevel.length,
              })}
            </p>
          )}
        </>
      ) : (
        <p className="text-sm text-control-light">
          {policy.bindings.length === 0
            ? t("sql-editor.saved-query-share.not-shared")
            : policy.bindings.flatMap((binding) => binding.members).join(", ")}
        </p>
      )}

      {canManage && policy.bindings.length === 0 && (
        <p className="text-xs text-control-light">
          {t("sql-editor.saved-query-share.private-hint")}
        </p>
      )}
    </section>
  );
}
