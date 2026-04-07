import { DATASOURCE_ADMIN_USER_NAME } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";

export type InfoSection =
  | "host"
  | "port"
  | "authentication"
  | "ssl"
  | "ssh"
  | "database";

export interface InfoSnippet {
  title: string;
  content: string;
  codeBlock?: {
    language: string;
    code: string;
  };
  learnMoreLinks?: {
    title: string;
    url: string;
  }[];
}

const postgresqlContent: Partial<Record<InfoSection, InfoSnippet>> = {
  host: {
    title: "PostgreSQL Host",
    content:
      "Enter the hostname or IP address of your PostgreSQL server. For cloud-hosted databases, use the endpoint provided by your cloud provider.",
    learnMoreLinks: [
      {
        title: "Connect Your Instance",
        url: "https://docs.bytebase.com/get-started/instance?source=console",
      },
    ],
  },
  authentication: {
    title: "PostgreSQL Authentication",
    content: `Create a dedicated user "${DATASOURCE_ADMIN_USER_NAME}" for Bytebase to manage your database. Run the following SQL as a superuser:`,
    codeBlock: {
      language: "sql",
      code: `CREATE USER ${DATASOURCE_ADMIN_USER_NAME} WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';

ALTER USER ${DATASOURCE_ADMIN_USER_NAME} WITH SUPERUSER;`,
    },
    learnMoreLinks: [
      {
        title: "Configure Database User",
        url: "https://docs.bytebase.com/get-started/instance?source=console#configure-a-database-user",
      },
    ],
  },
  ssl: {
    title: "SSL Connection",
    content:
      "Cloud-hosted PostgreSQL databases (AWS RDS, Google Cloud SQL, Azure Database) typically require SSL connections. Enable SSL and upload the CA certificate provided by your cloud provider.",
    learnMoreLinks: [
      {
        title: "SSL/TLS Connection",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssltls-connection",
      },
    ],
  },
  ssh: {
    title: "SSH Tunnel",
    content:
      "Use an SSH tunnel to connect to databases in private networks that are not directly accessible from Bytebase. This routes the connection through a bastion/jump host.",
    learnMoreLinks: [
      {
        title: "SSH Tunnel",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssh-tunnel",
      },
    ],
  },
};

const mysqlContent: Partial<Record<InfoSection, InfoSnippet>> = {
  host: {
    title: "MySQL Host",
    content:
      "Enter the hostname or IP address of your MySQL server. For cloud-hosted databases (AWS RDS, Google Cloud SQL), use the endpoint provided by your cloud provider.",
    learnMoreLinks: [
      {
        title: "Connect Your Instance",
        url: "https://docs.bytebase.com/get-started/instance?source=console",
      },
    ],
  },
  authentication: {
    title: "MySQL Authentication",
    content: `Create a dedicated user "${DATASOURCE_ADMIN_USER_NAME}" for Bytebase to manage your database. Run the following SQL:`,
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
        title: "Configure Database User",
        url: "https://docs.bytebase.com/get-started/instance?source=console#configure-a-database-user",
      },
    ],
  },
  ssl: {
    title: "SSL Connection",
    content:
      "Cloud-hosted MySQL databases (AWS RDS, Google Cloud SQL, Azure Database) typically require SSL connections. Enable SSL and upload the CA certificate provided by your cloud provider.",
    learnMoreLinks: [
      {
        title: "SSL/TLS Connection",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssltls-connection",
      },
    ],
  },
  ssh: {
    title: "SSH Tunnel",
    content:
      "Use an SSH tunnel to connect to databases in private networks that are not directly accessible from Bytebase. This routes the connection through a bastion/jump host.",
    learnMoreLinks: [
      {
        title: "SSH Tunnel",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssh-tunnel",
      },
    ],
  },
};

const mongodbContent: Partial<Record<InfoSection, InfoSnippet>> = {
  host: {
    title: "MongoDB Host",
    content:
      "Enter the hostname or IP address of your MongoDB server. For MongoDB Atlas, use the connection string host from your cluster's connect dialog.",
    learnMoreLinks: [
      {
        title: "Connect Your Instance",
        url: "https://docs.bytebase.com/get-started/instance?source=console",
      },
    ],
  },
  authentication: {
    title: "MongoDB Authentication",
    content: `Create a dedicated user "${DATASOURCE_ADMIN_USER_NAME}" for Bytebase in the admin database:`,
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
        title: "Configure Database User",
        url: "https://docs.bytebase.com/get-started/instance?source=console#configure-a-database-user",
      },
    ],
  },
  ssl: {
    title: "SSL Connection",
    content:
      "MongoDB Atlas and other cloud-hosted MongoDB services require SSL connections by default. Enable SSL and provide the CA certificate if needed.",
    learnMoreLinks: [
      {
        title: "SSL/TLS Connection",
        url: "https://docs.bytebase.com/get-started/instance?source=console#ssltls-connection",
      },
    ],
  },
  ssh: {
    title: "SSH Tunnel",
    content:
      "Use an SSH tunnel to connect to MongoDB instances in private networks. This is useful when the database is behind a firewall or in a VPC.",
    learnMoreLinks: [
      {
        title: "SSH Tunnel",
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
