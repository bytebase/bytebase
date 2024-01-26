ALTER TABLE "role" DISABLE TRIGGER update_role_updated_ts;

UPDATE "role"
SET permissions = '{
  "permissions": [
    "bb.changeHistories.get",
    "bb.changeHistories.list",
    "bb.databases.get",
    "bb.databases.getSchema",
    "bb.databases.list",
    "bb.issueComments.create",
    "bb.issues.get",
    "bb.issues.list",
    "bb.planCheckRuns.list",
    "bb.planCheckRuns.run",
    "bb.plans.get",
    "bb.plans.list",
    "bb.projects.get",
    "bb.projects.getIamPolicy",
    "bb.rollouts.get",
    "bb.taskRuns.list"
  ]
}'::JSONB;

ALTER TABLE "role" ENABLE TRIGGER update_role_updated_ts;
