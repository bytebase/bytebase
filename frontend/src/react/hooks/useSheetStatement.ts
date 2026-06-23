import { useEffect, useState } from "react";
import { useAppStore } from "@/react/stores/app";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { getStatementSize } from "@/utils/sheet";
import { extractSheetUID, getSheetStatement } from "@/utils/v1/sheet";

export type SheetStatementSnapshot = {
  statement: string;
  /** A fetch is in flight; render a loading state rather than an empty one. */
  isLoading: boolean;
  /** The cached statement is a preview of a larger sheet (size < contentSize). */
  isTruncated: boolean;
};

const EMPTY_SNAPSHOT: SheetStatementSnapshot = {
  statement: "",
  isLoading: false,
  isTruncated: false,
};

const snapshotOfSheet = (sheet: Sheet): SheetStatementSnapshot => {
  const statement = getSheetStatement(sheet);
  return {
    statement,
    isLoading: false,
    isTruncated: getStatementSize(statement) < sheet.contentSize,
  };
};

/**
 * Resolve a sheet's statement synchronously from what is already in memory, so
 * the first paint never shows an empty "No data" placeholder before the loader
 * has started. Local (draft) sheets are always in memory; a remote sheet is
 * used when cached, otherwise we report `isLoading` because a fetch will run.
 *
 * `getLocalSheet` is injected so this stays decoupled from any single page's
 * local-sheet store; callers that never use draft sheets (e.g. rollout tasks)
 * can omit it.
 */
export const seedSheetStatement = (
  sheetName: string,
  getLocalSheet?: (name: string) => Sheet | undefined
): SheetStatementSnapshot => {
  if (!sheetName) {
    return EMPTY_SNAPSHOT;
  }
  if (extractSheetUID(sheetName).startsWith("-")) {
    const localSheet = getLocalSheet?.(sheetName);
    return localSheet ? snapshotOfSheet(localSheet) : EMPTY_SNAPSHOT;
  }
  const cached = useAppStore.getState().getSheetByName(sheetName);
  return cached
    ? snapshotOfSheet(cached)
    : { statement: "", isLoading: true, isTruncated: false };
};

/**
 * Load a sheet's statement without the first-paint "No data" flash. State is
 * seeded synchronously from cache (see {@link seedSheetStatement}) and re-seeded
 * during render whenever the sheet changes — so switching specs/stages paints
 * the cached statement (or a loading state) immediately instead of flashing the
 * previous sheet's content or an empty placeholder, then a remote fetch fills in
 * any sheet that wasn't cached.
 *
 * The render-time re-seed keeps it correct under both call-site idioms: a
 * consumer may remount it per entity (`key={id}`) or reuse a single instance as
 * the `sheetName` prop changes. Owner components that keep their own mutable
 * statement reuse {@link seedSheetStatement} directly instead.
 */
export const useSheetStatement = ({
  enabled,
  sheetName,
  getLocalSheet,
}: {
  enabled: boolean;
  sheetName: string;
  getLocalSheet?: (name: string) => Sheet | undefined;
}): SheetStatementSnapshot => {
  const [snapshot, setSnapshot] = useState<SheetStatementSnapshot>(() =>
    enabled ? seedSheetStatement(sheetName, getLocalSheet) : EMPTY_SNAPSHOT
  );

  const seedKey = enabled ? sheetName : "";
  const [trackedSeedKey, setTrackedSeedKey] = useState(seedKey);
  if (seedKey !== trackedSeedKey) {
    setTrackedSeedKey(seedKey);
    setSnapshot(
      enabled ? seedSheetStatement(sheetName, getLocalSheet) : EMPTY_SNAPSHOT
    );
  }

  useEffect(() => {
    if (!enabled || !sheetName) {
      return;
    }
    // Local and already-cached sheets were seeded synchronously above.
    if (extractSheetUID(sheetName).startsWith("-")) {
      return;
    }
    if (useAppStore.getState().getSheetByName(sheetName)) {
      return;
    }

    let canceled = false;
    const load = async () => {
      try {
        const sheet = await useAppStore
          .getState()
          .getOrFetchSheetByName(sheetName);
        if (canceled || !sheet) {
          return;
        }
        setSnapshot(snapshotOfSheet(sheet));
      } finally {
        if (!canceled) {
          setSnapshot((prev) =>
            prev.isLoading ? { ...prev, isLoading: false } : prev
          );
        }
      }
    };
    void load();
    return () => {
      canceled = true;
    };
    // getLocalSheet is intentionally excluded: only the synchronous seed reads
    // it, and the effect handles remote sheets exclusively.
  }, [enabled, sheetName]);

  return snapshot;
};
