UPDATE instance SET metadata = REPLACE(metadata::text, 'clientSecretCredential', 'azureCredential')
::jsonb;