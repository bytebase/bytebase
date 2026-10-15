import { create } from "@bufbuild/protobuf";
import { useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormControlGroup, FormError, FormField } from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types";
import {
  PolicyResourceType,
  PolicyType,
  ReviewAIPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";

// The server's limit: the whole policy goes into every review prompt.
export const MAX_REVIEW_AI_POLICY_BYTES = 64 * 1024;

// The server rejects text that Go's strings.TrimSpace empties, which trims the
// Unicode White_Space set. JavaScript's trim differs: it keeps U+0085 and
// removes U+FEFF.
const isBlank = (content: string) => /^\p{White_Space}*$/u.test(content);

// The server measures the text in UTF-8 bytes, not characters.
const utf8Bytes = (content: string) => new TextEncoder().encode(content).length;

export interface ReviewAIPolicyDraft {
  // False until the policy has been read.
  loaded: boolean;
  readFailed: boolean;
  reload: () => void;
  // The policy text, or the unsaved edit; empty when there is no policy.
  content: string;
  setContent: (content: string) => void;
  // A policy paused through the API is kept but not read by the AI review.
  paused: boolean;
  canEdit: boolean;
  error: "too-large" | "cannot-clear" | "remove-only" | undefined;
  size: number;
  isDirty: boolean;
  saving: boolean;
  revert: () => void;
  // Resolves whether the policy saved. The page reports the update; a failure
  // is reported by the response middleware and keeps the draft.
  save: () => Promise<boolean>;
}

// The page owns the draft so the policy saves from the page's one sticky
// footer. Blank text saves as no policy, since the server rejects a blank one.
export function useReviewAIPolicyDraft({
  parentPath,
  resourceType,
  can,
  enabled = true,
}: {
  // The workspace or project the policy belongs to.
  parentPath: string;
  resourceType: PolicyResourceType;
  can: (permission: Permission) => boolean;
  enabled?: boolean;
}): ReviewAIPolicyDraft {
  const policy = useAppStore((state) =>
    state.getPolicyByParentAndType({
      parentPath,
      policyType: PolicyType.REVIEW_AI,
    })
  );
  const [draft, setDraft] = useState<string>();
  const [attempt, setAttempt] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const [readFailed, setReadFailed] = useState(false);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    if (!enabled || !parentPath) return;
    let active = true;
    // The route component is reused when moving to another project.
    setDraft(undefined);
    setLoaded(false);
    setReadFailed(false);
    void useAppStore
      .getState()
      .fetchPolicyByParentAndType({
        parentPath,
        policyType: PolicyType.REVIEW_AI,
        refresh: attempt > 0,
      })
      .then((result) => {
        if (!active) return;
        // A read that failed may leave an earlier policy in the cache, so the
        // result decides, not the cache.
        if (result === undefined) {
          setReadFailed(true);
          return;
        }
        setLoaded(true);
      });
    return () => {
      active = false;
    };
  }, [enabled, parentPath, attempt]);

  // A missing policy is cached as a stand-in without a payload.
  const stored = policy?.policy.case === "reviewAiPolicy" ? policy : undefined;
  const storedContent =
    policy?.policy.case === "reviewAiPolicy" ? policy.policy.value.content : "";
  const content = draft ?? storedContent;
  const isDirty =
    draft !== undefined &&
    (isBlank(draft) ? stored !== undefined : draft !== storedContent);
  const clearing = isDirty && isBlank(content);
  const size = utf8Bytes(content);
  // Clearing an existing policy deletes it and any other edit updates it, so
  // a role with only one of the two may make only that change.
  const error =
    isDirty && stored && !clearing && !can("bb.policies.update")
      ? "remove-only"
      : isDirty && size > MAX_REVIEW_AI_POLICY_BYTES
        ? "too-large"
        : clearing && !can("bb.policies.delete")
          ? "cannot-clear"
          : undefined;

  const save = async () => {
    if (!isDirty) return true;
    if (error) return false;
    setSaving(true);
    try {
      const store = useAppStore.getState();
      if (clearing && stored) {
        await store.deletePolicy(stored.name);
      } else {
        // The update mask names only the text, so a paused policy stays
        // paused; a new policy is created enforced.
        await store.upsertPolicy({
          parentPath,
          policy: {
            type: PolicyType.REVIEW_AI,
            resourceType,
            enforce: true,
            policy: {
              case: "reviewAiPolicy",
              value: create(ReviewAIPolicySchema, { content }),
            },
          },
        });
      }
      setDraft(undefined);
      return true;
    } catch {
      return false;
    } finally {
      setSaving(false);
    }
  };

  return {
    loaded,
    readFailed,
    reload: () => setAttempt((n) => n + 1),
    content,
    setContent: setDraft,
    paused: stored !== undefined && !stored.enforce,
    canEdit: stored
      ? can("bb.policies.update") || can("bb.policies.delete")
      : can("bb.policies.create"),
    error,
    size,
    isDirty,
    saving,
    revert: () => setDraft(undefined),
    save,
  };
}

export function ReviewAIPolicyField({
  policy,
  description,
}: {
  policy: ReviewAIPolicyDraft;
  description: string;
}) {
  const { t } = useTranslation();
  const id = useId();
  const titleId = `${id}-title`;
  const descriptionId = `${id}-description`;
  const errorId = `${id}-error`;

  if (policy.readFailed) {
    return (
      <Alert variant="error">
        <div className="flex flex-col items-start gap-y-3">
          {t("sql-review.ai-review.policy.load-failed")}
          <Button appearance="outline" size="sm" onClick={policy.reload}>
            {t("sql-review.standard-rules.retry")}
          </Button>
        </div>
      </Alert>
    );
  }

  return (
    <FormField
      title={<span id={titleId}>{t("sql-review.ai-review.policy.self")}</span>}
      description={<span id={descriptionId}>{description}</span>}
    >
      <FormControlGroup>
        {policy.loaded && policy.paused && (
          <Alert
            variant="warning"
            appearance="caption"
            description={t("sql-review.ai-review.policy.paused")}
          />
        )}
        {!policy.loaded ? (
          <p className="text-sm text-control-light">{t("common.loading")}</p>
        ) : policy.canEdit ? (
          <Textarea
            aria-labelledby={titleId}
            aria-describedby={
              policy.error ? `${descriptionId} ${errorId}` : descriptionId
            }
            aria-invalid={policy.error !== undefined}
            rows={8}
            placeholder={t("sql-review.ai-review.policy.placeholder")}
            value={policy.content}
            onChange={(event) => policy.setContent(event.target.value)}
            disabled={policy.saving}
          />
        ) : policy.content ? (
          <p className="whitespace-pre-wrap break-words rounded-sm border border-block-border px-3 py-2 text-sm text-main">
            {policy.content}
          </p>
        ) : (
          <p className="text-sm text-control-light">
            {t("sql-review.ai-review.policy.none")}
          </p>
        )}
      </FormControlGroup>
      {policy.error === "too-large" && (
        <FormError id={errorId}>
          {t("sql-review.ai-review.policy.too-large", {
            size: policy.size.toLocaleString(),
            limit: MAX_REVIEW_AI_POLICY_BYTES.toLocaleString(),
          })}
        </FormError>
      )}
      {policy.error === "cannot-clear" && (
        <FormError id={errorId}>
          {t("sql-review.ai-review.policy.cannot-clear")}
        </FormError>
      )}
      {policy.error === "remove-only" && (
        <FormError id={errorId}>
          {t("sql-review.ai-review.policy.remove-only")}
        </FormError>
      )}
    </FormField>
  );
}
