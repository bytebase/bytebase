// IAM related resource definition.
export interface DatabaseResource {
  instanceResourceId?: string;
  // the database full name
  databaseName: string;
  schema?: string;
  table?: string;
  column?: string;
}
