export function buildPlanCreatePrompt(): string {
  return "I am drafting a database change in the current Plan. Ask what change I want if I have not described it. Inspect the selected database's latest synced schema before proposing SQL. Return reviewable DDL and offer the action to apply it to this Plan. Do not create another Plan or execute SQL.";
}
