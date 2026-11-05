export function buildPlanCreatePrompt(): string {
  return "I am drafting a database change in the current Plan. Ask what change I want if I have not described it. Inspect the selected database's latest synced schema before proposing SQL. Return reviewable DDL and offer the action to apply it to this Plan. Do not create another Plan or execute SQL.";
}

export function buildPlanReviewPrompt(): string {
  return "Review this Plan and its linked Issue, if present. Inspect the SQL from its Sheets or Releases and current checks before drawing conclusions. Identify concrete correctness, safety, and rollout risks; separate verified evidence from unknowns. Do not modify the Plan or approve/reject anything.";
}

export function buildIssueReviewPrompt(): string {
  return "Review this Issue and its linked Plan and SQL when present. Use current Issue and Plan evidence, inspecting the source SQL and checks relevant to the change. Identify only specific, evidence-backed security or correctness concerns; distinguish verified facts from unknowns. If you find a specific, evidence-backed concern, prepare a comment draft, but never post it. Do not approve, reject, or execute anything.";
}
