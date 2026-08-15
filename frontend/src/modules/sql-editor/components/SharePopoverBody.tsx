import { Copy, Link2 } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { SQL_EDITOR_SAVED_QUERY_MODULE } from "@/app/router/handles";
import { Button } from "@/components/ui/button";
import { writeTextToClipboard } from "@/lib/clipboard";
import { useAppStore } from "@/stores/app";
import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import {
  extractProjectResourceName,
  extractSavedQueryID,
  isSavedQueryShareableV1,
} from "@/utils";

import { SavedQueryGrantEditor } from "./SavedQueryGrantEditor";

type Props = {
  readonly savedQuery?: SavedQuery;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/SharePopover.vue.
 * Renders the share popover body: the saved query's grants, plus its deep
 * link. The link carries location, not access — opening it still runs the
 * same read check — so the two are shown together but do different work.
 */
export function SharePopoverBody({ savedQuery }: Props) {
  const { t } = useTranslation();
  const workspaceExternalURL = useAppStore((s) => s.serverInfo?.externalUrl);
  const currentUser = useAppStore((s) => s.currentUser);

  // Sharing: the creator, or a project-level bb.savedQueries.setIamPolicy.
  // Both inputs are already on the client, so this needs no extra request.
  const canManage = useMemo(() => {
    if (!savedQuery || !currentUser) return false;
    return isSavedQueryShareableV1(savedQuery);
  }, [savedQuery, currentUser]);

  const sharedTabLink = useMemo(() => {
    if (!savedQuery) return "";
    const route = router.resolve({
      name: SQL_EDITOR_SAVED_QUERY_MODULE,
      params: {
        project: extractProjectResourceName(savedQuery.project),
        savedQuery: extractSavedQueryID(savedQuery.name),
      },
    });
    return new URL(route.href, workspaceExternalURL || window.location.origin)
      .href;
  }, [savedQuery, workspaceExternalURL]);

  const handleCopyLink = async () => {
    if (await writeTextToClipboard(sharedTabLink)) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } else {
      // clipboard not available
    }
  };

  return (
    <div className="w-96 p-2 flex flex-col gap-y-4">
      <section className="w-full flex flex-row justify-between items-center">
        <div className="pr-4">
          <h2 className="text-lg font-semibold">{t("common.share")}</h2>
        </div>
      </section>

      {savedQuery && (
        <SavedQueryGrantEditor savedQuery={savedQuery} canManage={canManage} />
      )}

      {/* Link input + copy button — single bordered container with rounded
          inner corners. No group-level focus ring; only the input shows a
          focus highlight. */}
      <div className="flex items-center h-8 rounded-xs border border-control-border overflow-hidden">
        {/* Link icon prefix (gray addon) */}
        <div className="flex items-center justify-center h-full px-2 bg-control-bg text-control-light border-r border-control-border">
          <Link2 className="size-5" />
        </div>
        {/* URL input — always read-only; the link itself is not editable. */}
        <input
          type="text"
          readOnly
          value={sharedTabLink}
          className="flex-1 min-w-0 h-full px-2 bg-background text-control text-sm cursor-text appearance-none border-0 shadow-none outline-hidden focus:outline-hidden focus:ring-0 focus:border-0 focus:shadow-none"
        />
        {/* Copy button — enabled whenever the shared savedQuery has a link.
            Gated on the shared savedQuery (sharedTabLink), NOT the current tab's
            status: the popover can be opened for any savedQuery from the tree,
            so the current tab's dirty state is irrelevant. Only an unsaved draft
            (no savedQuery → no link) disables it. */}
        <Button
          type="button"
          appearance="secondary"
          size="sm"
          data-copy-btn
          disabled={!sharedTabLink}
          onClick={handleCopyLink}
          className="h-full rounded-none border-l border-control-border bg-background enabled:hover:bg-control-bg-hover enabled:hover:text-main disabled:bg-control-bg focus-visible:ring-inset focus-visible:ring-offset-0"
        >
          <Copy className="size-4" />
        </Button>
      </div>
    </div>
  );
}
