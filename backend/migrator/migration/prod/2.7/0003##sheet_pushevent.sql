UPDATE
  sheet
SET
  payload = payload || json_build_object('vcsPayload', json_build_object('pushEvent', (task.payload->>'pushEvent')::JSONB))
FROM task
Where task.payload->>'pushEvent' IS NOT NULL AND task.payload->>'sheetId'::text = sheet.id::text ;
