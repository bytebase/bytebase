import { Loader2, RefreshCwIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { useCurrentUserV1 } from "@/store";
import { storageKeySqlEditorAiSuggestion } from "@/utils";
import { useDynamicSuggestions } from "../logic";
import { useAIContext } from "./context";

type Props = {
  readonly onEnter: (query: string) => void;
};

/**
 * React port of `plugins/ai/components/DynamicSuggestions.vue`.
 *
 * Shows up to one "suggested prompt" pill (LLM-generated) with refresh
 * / dismiss controls. Clicking the pill emits the suggestion to the
 * parent (`ChatPanel`) which submits it via `requestAI(query)`.
 *
 * `useDynamicSuggestions()` returns a Vue `ComputedRef<SuggestionContext>`
 * where the inner object is a Pinia/Vue `reactive(...)` — we read each
 * field via its own `useVueState` getter so React only re-renders when
 * the specific field changes.
 */
export function DynamicSuggestions({ onEnter }: Props) {
  const { t } = useTranslation();
  const { databaseMetadata, engine, schema } = useAIContext();
  // `useDynamicSuggestions` returns a Vue `ComputedRef<SuggestionContext>`
  // whose `metadata` getter reads our params. We pass them via
  // `MaybeRefOrGetter` getters so Vue's reactivity follows React's
  // useVueState-bridged values (the React `databaseMetadata` etc. are
  // already plain values that re-read on each call).
  const suggestionsRef = useDynamicSuggestions({
    databaseMetadata: () => databaseMetadata,
    engine: () => engine,
    schema: () => schema,
  });
  const currentUserRef = useCurrentUserV1();
  const currentUserEmail = useVueState(() => currentUserRef.value.email);

  const ready = useVueState(() => suggestionsRef.value?.ready ?? false);
  const state = useVueState<"LOADING" | "IDLE" | "ENDED">(
    () => suggestionsRef.value?.state ?? "IDLE"
  );
  const suggestionsCount = useVueState(
    () => suggestionsRef.value?.suggestions.length ?? 0
  );
  const current = useVueState(() => suggestionsRef.value?.current());

  // Kick off the initial fetch when the component mounts and the cache
  // is empty — matches the Vue `onMounted` block.
  useEffect(() => {
    const suggestion = suggestionsRef.value;
    if (suggestion && suggestion.suggestions.length === 0) {
      void suggestion.fetch();
    }
  }, [suggestionsRef]);

  // Per-user dismissable flag persisted to localStorage. Defaults to
  // visible. Same storage key the Vue version used (`useDynamicLocalStorage`
  // backed by `vueuse.useStorage`); we keep the key compatible so a user
  // who dismissed in Vue stays dismissed in React.
  const storageKey = useMemo(
    () => storageKeySqlEditorAiSuggestion(currentUserEmail),
    [currentUserEmail]
  );
  const [showSuggestion, setShowSuggestion] = useState<boolean>(() => {
    try {
      const raw = localStorage.getItem(storageKey);
      if (raw === null) return true;
      return JSON.parse(raw) as boolean;
    } catch {
      return true;
    }
  });
  useEffect(() => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(showSuggestion));
    } catch {
      // ignore
    }
  }, [storageKey, showSuggestion]);

  const show = !ready || suggestionsCount > 0 || state === "LOADING";
  if (!show) return null;

  const handleConsume = () => {
    const suggestion = suggestionsRef.value;
    if (!suggestion) return;
    const curr = suggestion.current();
    if (!curr) return;
    onEnter(curr);
    suggestion.consume();
  };

  const handleRefresh = () => {
    suggestionsRef.value?.consume();
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
