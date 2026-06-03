import { Loader2, RefreshCwIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { storageKeySqlEditorAiSuggestion } from "@/utils";
import { useAIContext } from "./context";
import { useDynamicSuggestions } from "./useDynamicSuggestions";

type Props = {
  readonly onEnter: (query: string) => void;
};

function loadShowSuggestion(key: string): boolean {
  try {
    const raw = localStorage.getItem(key);
    return raw === null ? true : (JSON.parse(raw) as boolean);
  } catch {
    return true;
  }
}

/**
 * React port of `plugins/ai/components/DynamicSuggestions.vue`.
 *
 * Shows up to one "suggested prompt" pill (LLM-generated) with refresh
 * / dismiss controls. Clicking the pill emits the suggestion to the
 * parent (`ChatPanel`) which submits it via `requestAI(query)`.
 *
 * `useDynamicSuggestions()` is a React hook backed by a module store
 * (subscribed via `useSyncExternalStore`); it returns the current
 * `SuggestionContext` (or undefined) and re-renders when the suggestions
 * or fetch state for the active metadata change.
 */
export function DynamicSuggestions({ onEnter }: Props) {
  const { t } = useTranslation();
  const { databaseMetadata, engine, schema } = useAIContext();
  const suggestion = useDynamicSuggestions({
    databaseMetadata,
    engine,
    schema,
  });
  const currentUserEmail = useCurrentUser().email;

  const ready = suggestion?.ready ?? false;
  const state = suggestion?.state ?? "IDLE";
  const suggestionsCount = suggestion?.suggestions.length ?? 0;
  const current = suggestion?.current();

  // Kick off the initial fetch when the component mounts and the cache
  // is empty — matches the Vue `onMounted` block. `useDynamicSuggestions`
  // returns a fresh `computed(...)` each render, so this effect re-runs on
  // every render — gate by `state` so we don't pile concurrent `fetch()`s
  // on top of an in-flight one (each fetch is a paid AI completion).
  useEffect(() => {
    if (
      suggestion &&
      suggestion.suggestions.length === 0 &&
      suggestion.state === "IDLE"
    ) {
      void suggestion.fetch();
    }
  }, [suggestion, state]);

  // Per-user dismissable flag persisted to localStorage. Defaults to
  // visible. Same storage key the Vue version used (`useDynamicLocalStorage`
  // backed by `vueuse.useStorage`); we keep the key compatible so a user
  // who dismissed in Vue stays dismissed in React.
  const storageKey = useMemo(
    () => storageKeySqlEditorAiSuggestion(currentUserEmail),
    [currentUserEmail]
  );
  // Keep the in-memory flag bound to a specific storage key so a key
  // change (user resolves / signs in as someone else) re-reads from the
  // new key *before* any persistence runs — otherwise the write effect
  // would clobber the new user's saved preference with the old user's
  // in-memory value.
  const [persisted, setPersisted] = useState<{ key: string; value: boolean }>(
    () => ({ key: storageKey, value: loadShowSuggestion(storageKey) })
  );
  useEffect(() => {
    if (persisted.key === storageKey) return;
    setPersisted({ key: storageKey, value: loadShowSuggestion(storageKey) });
  }, [storageKey, persisted.key]);
  useEffect(() => {
    if (persisted.key !== storageKey) return;
    try {
      localStorage.setItem(storageKey, JSON.stringify(persisted.value));
    } catch {
      // ignore
    }
  }, [storageKey, persisted.key, persisted.value]);
  const showSuggestion = persisted.value;
  const setShowSuggestion = (next: boolean) =>
    setPersisted({ key: storageKey, value: next });

  const show = !ready || suggestionsCount > 0 || state === "LOADING";
  if (!show) return null;

  const handleConsume = () => {
    if (!suggestion) return;
    const curr = suggestion.current();
    if (!curr) return;
    onEnter(curr);
    suggestion.consume();
  };

  const handleRefresh = () => {
    suggestion?.consume();
  };

  return (
    <div className="flex items-center overflow-hidden h-[22px]">
      {!ready && (
        <>
          <Loader2 className="mr-2 size-4 animate-spin" />
          <span className="text-sm">
            {t("plugin.ai.conversation.tips.suggest-prompt")}
          </span>
        </>
      )}

      {ready && showSuggestion && (
        <div className="relative flex items-center gap-1 overflow-hidden text-xs leading-4">
          {current && (
            <Button
              variant="outline"
              size="xs"
              className="flex-1 overflow-hidden h-[22px]"
              onClick={handleConsume}
            >
              <span className="w-full truncate leading-[22px]">{current}</span>
            </Button>
          )}

          {state === "LOADING" && (
            <Loader2 className="shrink-0 size-4 animate-spin" />
          )}
          {state === "IDLE" && (
            <div className="flex items-center">
              <Button
                variant="ghost"
                size="xs"
                className="shrink-0 h-[22px] px-1.5"
                onClick={handleRefresh}
                aria-label={t("plugin.ai.conversation.tips.suggest-prompt")}
              >
                <RefreshCwIcon className="size-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                className="shrink-0 h-[22px] px-1.5"
                onClick={() => setShowSuggestion(false)}
                aria-label={t("common.close")}
              >
                <XIcon className="size-3.5" />
              </Button>
            </div>
          )}
          {state === "ENDED" && (
            <span className="shrink-0 text-gray-500">
              {t("plugin.ai.conversation.tips.no-more")}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
