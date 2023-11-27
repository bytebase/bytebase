import { Action, CompareFn } from "@bytebase/vue-kbar";
import { findLastIndex } from "lodash-es";

const MAX_RANKING = Infinity;

export const ACTION_RANKINGS = [
  "bb.recently_visited.",
  "bb.quickaction.",
  "bb.navigation.",
  "bb.navigation.global.",
  "bb.project.",
  "bb.database.",
  "bb.preferences.",
];

export const compareAction: CompareFn = (a, b) => {
  const ar = getActionRankingById(a.item);
  const br = getActionRankingById(b.item);

  // Sort by original index if they have same ranks.
  if (ar === br) return a.index - b.index;

  // Otherwise sort by ranking.
  return ar - br;
};

function getActionRankingById(action: Action) {
  const rank = findLastIndex(ACTION_RANKINGS, (prefix) =>
    action.id.startsWith(prefix)
  );
  // non-specified namespaces should always rank behind specified ones.
  if (rank < 0) return MAX_RANKING;

  return rank;
}
