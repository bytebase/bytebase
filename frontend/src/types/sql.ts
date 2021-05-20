export type DBType = "MYSQL";

export type SqlConfig = {
  dbType: DBType;
  host: string;
  port?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type SqlResultSet = {
  error: string;
};
