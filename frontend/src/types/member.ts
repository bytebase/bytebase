// IAM related resource definition.
export interface DatabaseResource {
  // the database full name
  databaseName: string;
  schema?: string;
  table?: string;
  column?: string;
}
