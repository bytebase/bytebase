UPDATE 
  project_member 
SET 
  condition = jsonb_set(
    "condition"::jsonb, 
    '{expression}', 
    REPLACE(
      REPLACE(
        REPLACE(
          REPLACE(
            REPLACE(
              (condition -> 'expression'):: TEXT, 
              'request.export_format == \"JSON\"', 
              'true'
            ), 
            'request.export_format == \"XLSX\"', 
            'true'
          ), 
          'request.export_format == \"SQL\"', 
          'true'
        ), 
        'request.export_format == \"CSV\"', 
        'true'
      ), 
      'request.row_limit == ', 
      'request.row_limit <= '
    )::JSONB
  ) 
WHERE role = 'EXPORTER' AND condition ? 'expression';
