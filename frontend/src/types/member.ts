// IAM related resource definition.
export interface DatabaseResource {
  instanceResourceId?: string;
  databaseResourceId?: string;
  // the database full name
  databaseFullName: string;
  schema?: string;
  table?: string;
  columns?: string[];
}
