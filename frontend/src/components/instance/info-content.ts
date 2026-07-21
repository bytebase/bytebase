import { DATASOURCE_ADMIN_USER_NAME } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";

export type InfoSection =
  | "host"
  | "port"
  | "authentication"
  | "ssl"
  | "ssh"
  | "database";

export type InfoSnippetContentKey =
  | "instance.info.mongodb.authentication.content"
  | "instance.info.mongodb.host.content"
  | "instance.info.mongodb.ssh.content"
  | "instance.info.mongodb.ssl.content"
  | "instance.info.mysql.authentication.content"
  | "instance.info.mysql.host.content"
  | "instance.info.mysql.ssh.content"
  | "instance.info.mysql.ssl.content"
  | "instance.info.postgresql.authentication.content"
  | "instance.info.postgresql.host.content"
  | "instance.info.postgresql.ssh.content"
  | "instance.info.postgresql.ssl.content";

export type InfoSnippetLinkTitleKey =
  | "instance.info.configure-database-user.link"
  | "instance.info.connect-instance.link"
  | "instance.info.ssh-tunnel.link"
  | "instance.info.ssl-tls-connection.link";

export type InfoSnippet = {
  contentKey: InfoSnippetContentKey;
  contentInterpolation?: Record<string, string>;
  codeBlock?: {
    language: string;
    code: string;
  };
  learnMoreLinks?: {
    titleKey: InfoSnippetLinkTitleKey;
    url: string;
  }[];
};

const postgresqlContent: Partial<Record<InfoSection, InfoSnippet>> = {
  host: {
    contentKey: "instance.info.postgresql.host.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.connect-instance.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console",
      },
    ],
  },
  authentication: {
    contentKey: "instance.info.postgresql.authentication.content",
    contentInterpolation: {
      user: DATASOURCE_ADMIN_USER_NAME,
    },
    codeBlock: {
      language: "sql",
      code: `CREATE USER ${DATASOURCE_ADMIN_USER_NAME} WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';

ALTER USER ${DATASOURCE_ADMIN_USER_NAME} WITH SUPERUSER;`,
    },
    learnMoreLinks: [
      {
        titleKey: "instance.info.configure-database-user.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#configure-a-database-user",
      },
    ],
  },
  ssl: {
    contentKey: "instance.info.postgresql.ssl.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.ssl-tls-connection.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssltls-connection",
      },
    ],
  },
  ssh: {
    contentKey: "instance.info.postgresql.ssh.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.ssh-tunnel.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssh-tunnel",
      },
    ],
  },
};

const mysqlContent: Partial<Record<InfoSection, InfoSnippet>> = {
  host: {
    contentKey: "instance.info.mysql.host.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.connect-instance.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console",
      },
    ],
  },
  authentication: {
    contentKey: "instance.info.mysql.authentication.content",
    contentInterpolation: {
      user: DATASOURCE_ADMIN_USER_NAME,
    },
    codeBlock: {
      language: "sql",
      code: `CREATE USER ${DATASOURCE_ADMIN_USER_NAME}@'%' IDENTIFIED BY 'YOUR_DB_PWD';

GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW,
DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES,
SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE,
RELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE
/*!80000 , SET_USER_ID */
ON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`,
    },
    learnMoreLinks: [
      {
        titleKey: "instance.info.configure-database-user.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#configure-a-database-user",
      },
    ],
  },
  ssl: {
    contentKey: "instance.info.mysql.ssl.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.ssl-tls-connection.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssltls-connection",
      },
    ],
  },
  ssh: {
    contentKey: "instance.info.mysql.ssh.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.ssh-tunnel.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssh-tunnel",
      },
    ],
  },
};

const mongodbContent: Partial<Record<InfoSection, InfoSnippet>> = {
  host: {
    contentKey: "instance.info.mongodb.host.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.connect-instance.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console",
      },
    ],
  },
  authentication: {
    contentKey: "instance.info.mongodb.authentication.content",
    contentInterpolation: {
      user: DATASOURCE_ADMIN_USER_NAME,
    },
    codeBlock: {
      language: "javascript",
      code: `use admin;
db.createUser({
  user: "${DATASOURCE_ADMIN_USER_NAME}",
  pwd: "YOUR_DB_PWD",
  roles: [
    {role: "readWriteAnyDatabase", db: "admin"},
    {role: "dbAdminAnyDatabase", db: "admin"},
    {role: "userAdminAnyDatabase", db: "admin"}
  ]
});`,
    },
    learnMoreLinks: [
      {
        titleKey: "instance.info.configure-database-user.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#configure-a-database-user",
      },
    ],
  },
  ssl: {
    contentKey: "instance.info.mongodb.ssl.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.ssl-tls-connection.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssltls-connection",
      },
    ],
  },
  ssh: {
    contentKey: "instance.info.mongodb.ssh.content",
    learnMoreLinks: [
      {
        titleKey: "instance.info.ssh-tunnel.link",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssh-tunnel",
      },
    ],
  },
};

const engineContentMap: Partial<
  Record<Engine, Partial<Record<InfoSection, InfoSnippet>>>
> = {
  [Engine.POSTGRES]: postgresqlContent,
  [Engine.MYSQL]: mysqlContent,
  [Engine.TIDB]: mysqlContent,
  [Engine.MARIADB]: mysqlContent,
  [Engine.OCEANBASE]: mysqlContent,
  [Engine.MONGODB]: mongodbContent,
};

export const getInfoContent = (
  engine: Engine,
  section: InfoSection
): InfoSnippet | undefined => {
  return engineContentMap[engine]?.[section];
};

export const hasInfoContent = (engine: Engine, section: InfoSection): boolean =>
  !!getInfoContent(engine, section);
