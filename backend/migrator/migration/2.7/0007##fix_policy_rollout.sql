ALTER TABLE policy DISABLE TRIGGER update_policy_updated_ts;

UPDATE policy
SET payload = '{
  "value": "MANUAL_APPROVAL_ALWAYS",
  "assigneeGroupList": [
    {
      "value": "PROJECT_OWNER",
      "issueType": "bb.issue.database.schema.update"
    },
    {
      "value": "PROJECT_OWNER",
      "issueType": "bb.issue.database.data.update"
    },
    {
      "value": "PROJECT_OWNER",
      "issueType": "bb.issue.database.schema.update.ghost"
    },
    {
      "value": "PROJECT_OWNER",
      "issueType": "bb.issue.database.general"
    }
  ]
}'
WHERE type = 'bb.policy.pipeline-approval'
AND payload @> '{
  "value": "MANUAL_APPROVAL_ALWAYS",
  "assigneeGroupList": [
    {
      "value": "PROJECT_OWNER",
      "issueType": "bb.issue.database.schema.update"
    }
  ]
}';

UPDATE policy
SET payload = '{
  "value": "MANUAL_APPROVAL_ALWAYS",
  "assigneeGroupList": [
    {
      "value": "WORKSPACE_OWNER_OR_DBA",
      "issueType": "bb.issue.database.schema.update"
    },
    {
      "value": "WORKSPACE_OWNER_OR_DBA",
      "issueType": "bb.issue.database.data.update"
    },
    {
      "value": "WORKSPACE_OWNER_OR_DBA",
      "issueType": "bb.issue.database.schema.update.ghost"
    },
    {
      "value": "WORKSPACE_OWNER_OR_DBA",
      "issueType": "bb.issue.database.general"
    }
  ]
}'
WHERE type = 'bb.policy.pipeline-approval'
AND payload @> '{
  "value": "MANUAL_APPROVAL_ALWAYS",
  "assigneeGroupList": [
    {
      "value": "WORKSPACE_OWNER_OR_DBA",
      "issueType": "bb.issue.database.schema.update"
    }
  ]
}';

ALTER TABLE policy ENABLE TRIGGER update_policy_updated_ts;
