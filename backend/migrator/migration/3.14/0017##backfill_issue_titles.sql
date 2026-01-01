-- Backfill issue titles and descriptions from their associated plans.
-- Issues created from plans previously had empty titles/descriptions and relied on COALESCE.
-- This migration makes the relationship explicit by copying plan data into issue records.
UPDATE issue
SET
    name = plan.name,
    description = plan.description
FROM plan
WHERE issue.plan_id = plan.id
  AND issue.name = '';
