UPDATE
  sheet
SET
  payload = payload || jsonb_build_object('vcsPayload', jsonb_build_object('pushEvent', (task.payload->>'pushEvent')::JSONB))
FROM task
Where task.payload->>'pushEvent' IS NOT NULL AND task.payload->>'sheetId'::text = sheet.id::text ;
