// IAM related resource definition.
export interface DatabaseResource {
  databaseName: string;
  schema?: string;
  table?: string;
  column?: string;
}
