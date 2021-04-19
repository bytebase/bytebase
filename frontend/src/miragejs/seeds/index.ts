import faker from "faker";
import environment from "../../store/modules/environment";
import {
  Database,
  Environment,
  Instance,
  Principal,
  StageType,
  Issue,
  DEFAULT_PROJECT_ID,
  ALL_DATABASE_NAME,
  IssueNew,
  IssueStatus,
  StageNew,
  StageStatus,
  StageId,
  DatabaseId,
  ProjectId,
  PrincipalId,
  IssueId,
  StepStatus,
} from "../../types";
import { databaseSlug, issueSlug } from "../../utils";

/*
 * Mirage JS guide on Seeds: https://miragejs.com/docs/data-layer/factories#in-development
 */
const workspacesSeeder = (server: any) => {
  // Workspace id is ALWAYS 1 for on-premise deployment
  const workspace1 = server.schema.workspaces.find(1);

  // Workspace 2 is just for verifying we are not returning
  // resources from different workspaces.
  const workspace2 = server.schema.workspaces.find(101);

  // User
  const ws1Owner = server.schema.users.find(1);
  const ws1DBA = server.schema.users.find(2);
  const ws1Dev1 = server.schema.users.find(3);
  const ws1Dev2 = server.schema.users.find(5);

  const ws1UserList = [ws1Owner, ws1DBA, ws1Dev1, ws1Dev2];

  const ws2DBA = server.schema.users.find(4);
  const ws2Dev = server.schema.users.find(1);
  const ws2UserList = [ws2DBA, ws2Dev];

  // Environment
  const environmentList1 = [];
  for (let i = 0; i < 5; i++) {
    environmentList1.push(
      server.create("environment", {
        workspace: workspace1,
        creatorId: ws1Owner.id,
        updaterId: ws1Owner.id,
      })
    );
  }
  workspace1.update({ environment: environmentList1 });

  const environmentList2 = [];
  for (let i = 0; i < 5; i++) {
    environmentList2.push(
      server.create("environment", {
        workspace: workspace2,
        creatorId: ws2DBA.id,
        updaterId: ws2DBA.id,
      })
    );
  }
  workspace2.update({ environment: environmentList2 });

  // Project
  const projectList1 = [];
  for (let i = 0; i < 5; i++) {
    projectList1.push(
      server.create("project", {
        workspace: workspace1,
        rowStatus: i < 4 ? "NORMAL" : "ARCHIVED",
        creatorId: ws1Dev1.id,
        updaterId: ws1Dev1.id,
      })
    );

    server.create("projectMember", {
      workspace: workspace1,
      project: projectList1[i],
      creatorId: ws1DBA.id,
      updaterId: ws1DBA.id,
      role: "OWNER",
      principalId: ws1DBA.id,
    });

    const userId = i % 2 == 0 ? ws1Owner.id : ws1Dev1.id;
    server.create("projectMember", {
      workspace: workspace1,
      project: projectList1[i],
      creatorId: ws1Owner.id,
      updaterId: ws1Owner.id,
      role: "DEVELOPER",
      principalId: userId,
    });
  }
  workspace1.update({ project: projectList1 });

  const projectList2 = [];
  for (let i = 0; i < 1; i++) {
    projectList2.push(
      server.create("project", {
        workspace: workspace2,
        creatorId: ws2Dev.id,
        updaterId: ws2Dev.id,
      })
    );

    server.create("projectMember", {
      workspace: workspace2,
      project: projectList2[i],
      creatorId: ws2DBA.id,
      updaterId: ws2DBA.id,
      role: "OWNER",
      principalId: ws2DBA.id,
    });
  }
  workspace2.update({ project: projectList2 });

  // Instance
  const instanceList1 = createInstanceList(
    server,
    workspace1.id,
    environmentList1,
    projectList1,
    ws1DBA,
    DEFAULT_PROJECT_ID
  );

  const instanceList2 = createInstanceList(
    server,
    workspace2.id,
    environmentList2,
    projectList2,
    ws2DBA,
    "2"
  );

  // Database
  const databaseList1 = [];
  databaseList1.push(
    server.schema.databases.where((item: any) => {
      return item.name != "*" && item.instanceId == instanceList1[0].id;
    }).models[0]
  );
  databaseList1.push(
    server.schema.databases.where((item: any) => {
      return item.name != "*" && item.instanceId == instanceList1[1].id;
    }).models[0]
  );
  databaseList1.push(
    server.schema.databases.where((item: any) => {
      return item.name != "*" && item.instanceId == instanceList1[2].id;
    }).models[0]
  );
  databaseList1.push(
    server.schema.databases.where((item: any) => {
      return item.name != "*" && item.instanceId == instanceList1[3].id;
    }).models[0]
  );
  databaseList1.push(
    server.schema.databases.where((item: any) => {
      return item.name != "*" && item.instanceId == instanceList1[4].id;
    }).models[0]
  );

  const databaseList2 = [];
  databaseList2.push(
    server.schema.databases.where((item: any) => {
      return item.name != "*" && item.instanceId == instanceList2[0].id;
    }).models[0]
  );

  // Issue
  let issue = server.create("issue", {
    type: "bytebase.general",
    name: "Hello, World!",
    description:
      "Welcome to Bytebase, this is the issue interface where DBAs and developers collaborate on database management issues such as: \n\n - Requesting a new database\n - Creating a table\n - Creating an index\n - Adding a column\n - Troubleshooting performance issue\n\nLet's bookmark this issue by clicking the star icon on the top of this page.",
    sql:
      "SELECT 'Welcome'\nFROM engineering\nWHERE role IN ('DBA', 'Developer') AND taste = 'Good';",
    creatorId: ws1Dev1.id,
    updaterId: ws1Dev1.id,
    assigneeId: ws1Owner.id,
    subscriberIdList: [ws1DBA.id, ws1Dev2.id, ws1Dev1.id, ws1Owner.id],
    project: projectList1[0],
    workspace: workspace1,
  });

  const createdActivity = server.create("activity", {
    actionType: "bytebase.issue.create",
    containerId: issue.id,
    creatorId: ws1Dev1.id,
    updaterId: ws1Dev1.id,
    workspace: workspace1,
  });

  for (let i = 0; i < 5; i++) {
    if (i % 2 == 0) {
      server.create("message", {
        type: "bb.msg.issue.comment",
        containerId: issue.id,
        creatorId:
          ws1UserList[Math.floor(Math.random() * ws1UserList.length)].id,
        receiverId: ws1Owner.id,
        workspace: workspace1,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          commentId: createdActivity.id,
        },
      });
    } else {
      server.create("message", {
        type: "bb.msg.issue.status.update",
        containerId: issue.id,
        creatorId:
          ws1UserList[Math.floor(Math.random() * ws1UserList.length)].id,
        receiverId: ws1Owner.id,
        workspace: workspace1,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          oldStatus: "OPEN",
          newStatus: "DONE",
        },
      });
    }
  }

  for (let i = 0; i < 5; i++) {
    if (i % 2 == 0) {
      server.create("message", {
        type: "bb.msg.issue.comment",
        containerId: issue.id,
        creatorId:
          ws1UserList[Math.floor(Math.random() * ws1UserList.length)].id,
        receiverId: ws1DBA.id,
        workspace: workspace1,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          commentId: createdActivity.id,
        },
      });
    } else {
      server.create("message", {
        type: "bb.msg.issue.status.update",
        containerId: issue.id,
        creatorId:
          ws1UserList[Math.floor(Math.random() * ws1UserList.length)].id,
        receiverId: ws1DBA.id,
        workspace: workspace1,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          oldStatus: "OPEN",
          newStatus: "CANCELED",
        },
      });
    }
  }

  for (let i = 0; i < 5; i++) {
    if (i % 2 == 0) {
      server.create("message", {
        type: "bb.msg.issue.comment",
        containerId: issue.id,
        creatorId:
          ws1UserList[Math.floor(Math.random() * ws1UserList.length)].id,
        receiverId: ws1Dev1.id,
        workspace: workspace1,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          commentId: createdActivity.id,
        },
      });
    } else {
      server.create("message", {
        type: "bb.msg.issue.status.update",
        containerId: issue.id,
        creatorId:
          ws1UserList[Math.floor(Math.random() * ws1UserList.length)].id,
        receiverId: ws1Dev1.id,
        workspace: workspace1,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          oldStatus: "OPEN",
          newStatus: "DONE",
        },
      });
    }
  }

  for (let i = 0; i < 5; i++) {
    if (i % 2 == 0) {
      server.create("message", {
        type: "bb.msg.issue.comment",
        containerId: issue.id,
        creatorId: ws2DBA.id,
        receiverId: ws2Dev.id,
        workspace: workspace2,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          commentId: createdActivity.id,
        },
      });
    } else {
      server.create("message", {
        type: "bb.msg.issue.status.update",
        containerId: issue.id,
        creatorId:
          ws1UserList[Math.floor(Math.random() * ws1UserList.length)].id,
        receiverId: ws2Dev.id,
        workspace: workspace1,
        payload: {
          issueName: faker.fake("{{lorem.sentence}}"),
          oldStatus: "OPEN",
          newStatus: "DONE",
        },
      });
    }
  }

  for (let i = 0; i < 3; i++) {
    const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
    server.create("activity", {
      actionType: "bytebase.issue.comment.create",
      containerId: issue.id,
      creatorId: user.id,
      updaterId: user.id,
      comment: faker.fake("{{lorem.paragraph}}"),
      workspace: workspace1,
    });
  }

  const tableNameList = [
    "warehouse",
    "customer",
    "order",
    "item",
    "stock",
    "history",
  ];

  issue = server.create("issue", {
    type: "bytebase.database.create",
    name: `Create database '${databaseList1[1].name}' for environment - ${environmentList1[1].name}`,
    creatorId: ws1Dev1.id,
    updaterId: ws1Dev1.id,
    assigneeId: ws1Owner.id,
    subscriberIdList: [ws1DBA.id, ws1Dev2.id],
    payload: {
      5: projectList1[1].id,
      6: environmentList1[1].id,
      8: databaseList1[1].name,
    },
    project: projectList1[1],
    workspace: workspace1,
  });

  const stage = server.create("stage", {
    creatorId: ws1Dev1.id,
    updaterId: ws1Dev1.id,
    name: "Create database",
    type: "bytebase.stage.database.create",
    status: "PENDING",
    databaseId: databaseList1[1].id,
    issue,
    workspace: workspace1,
  });

  server.create("step", {
    creatorId: ws1Dev1.id,
    updaterId: ws1Dev1.id,
    name: "Waiting approval",
    type: "bytebase.step.approve",
    status: "PENDING",
    issue,
    stage,
    workspace: workspace1,
  });

  server.create("activity", {
    actionType: "bytebase.issue.create",
    containerId: issue.id,
    creatorId: ws1Dev1.id,
    updaterId: ws1Dev1.id,
    workspace: workspace1,
  });

  for (let i = 0; i < 3; i++) {
    const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
    server.create("activity", {
      actionType: "bytebase.issue.comment.create",
      containerId: issue.id,
      creatorId: user.id,
      updaterId: user.id,
      comment: faker.fake("{{lorem.paragraph}}"),
      workspace: workspace1,
    });
  }

  type SQLData = {
    title: string;
    sql: string;
  };
  const randomUpdateSchemaIssueName = (): SQLData => {
    const tableName =
      tableNameList[Math.floor(Math.random() * tableNameList.length)];
    const list: SQLData[] = [
      {
        title: "Create table " + tableName,
        sql: `CREATE TABLE ${tableName} (\n  id INT NOT NULL,\n  name TEXT,\n  age INT,\n  PRIMARY KEY (name)\n);`,
      },
      {
        title: "Add index to " + tableName,
        sql: `CREATE INDEX ${tableName}_idx\nON ${tableName} (name);`,
      },
      {
        title: "Drop index from " + tableName,
        sql: `ALTER TABLE ${tableName}\nDROP INDEX ${tableName}_idx;`,
      },
      {
        title: "Add column to " + tableName,
        sql: `ALTER TABLE ${tableName}\nADD email VARCHAR(255);`,
      },
      {
        title: "Drop column from " + tableName,
        sql: `ALTER TABLE ${tableName}\nDROP COLUMN email;`,
      },
      {
        title: "Alter column to " + tableName,
        sql: `ALTER TABLE ${tableName}\nMODIFY COLUMN email TEXT;`,
      },
      {
        title: "Add foreign key to " + tableName,
        sql: `ALTER TABLE ${tableName}\nADD CONSTRAINT FK_${tableName}\nFOREIGN KEY (id) REFERENCES ${tableName}(ID);`,
      },
      {
        title: "Drop foreign key from " + tableName,
        sql: `ALTER TABLE ${tableName}\nDROP FOREIGN KEY FK_${tableName};`,
      },
    ];

    return list[Math.floor(Math.random() * list.length)];
  };

  const statusSetList: {
    issueStatus: IssueStatus;
    stageStatusList: StageStatus[];
    stepStatusList: StepStatus[];
  }[] = [
    {
      issueStatus: "OPEN",
      stageStatusList: ["PENDING"],
      stepStatusList: ["PENDING"],
    },
    {
      issueStatus: "OPEN",
      stageStatusList: ["DONE", "DONE", "RUNNING", "PENDING"],
      stepStatusList: ["DONE", "DONE", "RUNNING", "PENDING"],
    },
    {
      issueStatus: "DONE",
      stageStatusList: ["DONE", "SKIPPED", "DONE", "DONE"],
      stepStatusList: ["DONE", "CANCELED", "DONE", "DONE"],
    },
    {
      issueStatus: "OPEN",
      stageStatusList: ["DONE", "FAILED", "PENDING", "PENDING"],
      stepStatusList: ["DONE", "FAILED", "PENDING", "PENDING"],
    },
    {
      issueStatus: "CANCELED",
      stageStatusList: ["DONE", "SKIPPED", "DONE", "PENDING"],
      stepStatusList: ["DONE", "CANCELED", "DONE", "PENDING"],
    },
  ];

  for (let i = 0; i < 3; i++) {
    const data = randomUpdateSchemaIssueName();
    const statusSet =
      statusSetList[Math.floor(Math.random() * statusSetList.length)];
    issue = server.create("issue", {
      name: data.title,
      type: "bytebase.database.schema.update",
      creatorId: ws1Dev1.id,
      updaterId: ws1Dev1.id,
      assigneeId: ws1Owner.id,
      sql: data.sql,
      subscriberIdList: [ws1DBA.id, ws1Dev2.id],
      status: statusSet.issueStatus,
      project: projectList1[i],
      workspace: workspace1,
    });

    createUpdateSchemaStage(
      server,
      workspace1.id,
      issue.id,
      ws1Dev1.id,
      environmentList1,
      databaseList1,
      statusSet.stageStatusList,
      statusSet.stepStatusList
    );

    server.create("activity", {
      actionType: "bytebase.issue.create",
      containerId: issue.id,
      creatorId: ws1Dev1.id,
      updaterId: ws1Dev1.id,
      workspace: workspace1,
    });

    for (let i = 0; i < 3; i++) {
      const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
      server.create("activity", {
        actionType: "bytebase.issue.comment.create",
        containerId: issue.id,
        creatorId: user.id,
        updaterId: user.id,
        comment: faker.fake("{{lorem.paragraph}}"),
        workspace: workspace1,
      });
    }
  }

  for (let i = 0; i < 3; i++) {
    const data = randomUpdateSchemaIssueName();
    const statusSet =
      statusSetList[Math.floor(Math.random() * statusSetList.length)];
    issue = server.create("issue", {
      name: data.title,
      type: "bytebase.database.schema.update",
      creatorId: ws1Owner.id,
      updaterId: ws1Owner.id,
      assigneeId: ws1DBA.id,
      sql: data.sql,
      subscriberIdList: [ws1Dev2.id],
      status: statusSet.issueStatus,
      project: projectList1[i],
      workspace: workspace1,
    });

    createUpdateSchemaStage(
      server,
      workspace1.id,
      issue.id,
      ws1Owner.id,
      environmentList1,
      databaseList1,
      statusSet.stageStatusList,
      statusSet.stepStatusList
    );

    server.create("activity", {
      actionType: "bytebase.issue.create",
      containerId: issue.id,
      creatorId: ws1Owner.id,
      updaterId: ws1Owner.id,
      workspace: workspace1,
    });

    for (let i = 0; i < 3; i++) {
      const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
      server.create("activity", {
        actionType: "bytebase.issue.comment.create",
        containerId: issue.id,
        creatorId: user.id,
        updaterId: user.id,
        comment: faker.fake("{{lorem.paragraph}}"),
        workspace: workspace1,
      });
    }
  }

  for (let i = 0; i < 3; i++) {
    const data = randomUpdateSchemaIssueName();
    const statusSet =
      statusSetList[Math.floor(Math.random() * statusSetList.length)];
    issue = server.create("issue", {
      name: data.title,
      type: "bytebase.database.schema.update",
      creatorId: ws1Dev2.id,
      updaterId: ws1Dev2.id,
      assigneeId: ws1DBA.id,
      sql: data.sql,
      subscriberIdList: [ws1Owner.id, ws1Dev1.id],
      status: statusSet.issueStatus,
      project: projectList1[i],
      workspace: workspace1,
    });

    createUpdateSchemaStage(
      server,
      workspace1.id,
      issue.id,
      ws1Dev2.id,
      environmentList1,
      databaseList1,
      statusSet.stageStatusList,
      statusSet.stepStatusList
    );

    server.create("activity", {
      actionType: "bytebase.issue.create",
      containerId: issue.id,
      creatorId: ws1Dev2.id,
      updaterId: ws1Dev2.id,
      workspace: workspace1,
    });

    for (let i = 0; i < 3; i++) {
      const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
      server.create("activity", {
        actionType: "bytebase.issue.comment.create",
        containerId: issue.id,
        creatorId: user.id,
        updaterId: user.id,
        comment: faker.fake("{{lorem.paragraph}}"),
        workspace: workspace1,
      });
    }
  }

  const data = randomUpdateSchemaIssueName();
  const statusSet =
    statusSetList[Math.floor(Math.random() * statusSetList.length)];
  issue = server.create("issue", {
    name: data.title,
    type: "bytebase.database.schema.update",
    creatorId: ws2Dev.id,
    updaterId: ws2Dev.id,
    assigneeId: ws2DBA.id,
    sql: data.sql,
    status: statusSet.issueStatus,
    project: projectList2[0],
    workspace: workspace2,
  });

  createUpdateSchemaStage(
    server,
    workspace2.id,
    issue.id,
    ws2Dev.id,
    environmentList2,
    databaseList2,
    ["PENDING"],
    ["PENDING"]
  );

  server.create("activity", {
    actionType: "bytebase.issue.create",
    containerId: issue.id,
    creatorId: ws2Dev.id,
    updaterId: ws2Dev.id,
    workspace: workspace2,
  });

  // Bookmark
  {
    // Workspace 1
    // db
    server.create("bookmark", {
      workspace: workspace1,
      name: databaseList1[0].name,
      link: `/db/${databaseSlug(databaseList1[0])}`,
      creatorId: ws1Owner.id,
    });
  }

  {
    // Workspace 2
    // Issue
    const issue = server.schema.issues.findBy({
      workspaceId: workspace2.id,
    });
    server.create("bookmark", {
      workspace: workspace2,
      name: issue.name,
      link: `/issue/${issueSlug(issue.name, issue.id)}`,
      creatorId: ws2DBA.id,
    });
  }
};

const createInstanceList = (
  server: any,
  workspaceId: string,
  enviromentList: { id: string }[],
  projectList: { id: string }[],
  dba: { id: string },
  defaultProjectId: ProjectId
): Instance[] => {
  const instanceNamelist = [
    "On-premise MySQL instance",
    "AWS RDS instance",
    "GCP Cloud SQL instance",
    "Azure SQL instance",
    "AliCloud RDS instance",
  ];

  const instanceList = [];
  for (let i = 0; i < 5; i++) {
    const instance = server.create("instance", {
      workspaceId: workspaceId,
      name:
        instanceNamelist[Math.floor(Math.random() * instanceNamelist.length)] +
        (i + 1),
      // Create an extra instance for prod.
      environmentId: i == 4 ? enviromentList[3].id : enviromentList[i].id,
      creatorId: dba.id,
      updaterId: dba.id,
    });
    instanceList.push(instance);

    const allDatabase = server.create("database", {
      name: ALL_DATABASE_NAME,
      workspaceId: workspaceId,
      projectId: defaultProjectId,
      instance,
      creatorId: dba.id,
      updaterId: dba.id,
    });

    server.create("dataSource", {
      workspaceId: instance.workspaceId,
      instance,
      database: allDatabase,
      creatorId: dba.id,
      updaterId: dba.id,
      name: instance.name + " admin data source",
      type: "ADMIN",
      username: "adminRW",
      password: "pwdadminRW",
    });

    for (let j = 0; j < 2; j++) {
      server.create("database", {
        workspaceId: workspaceId,
        projectId: projectList[i % projectList.length].id,
        instance,
        creatorId: dba.id,
        updaterId: dba.id,
      });
    }
  }

  server.create("database", {
    workspaceId: workspaceId,
    projectId: DEFAULT_PROJECT_ID,
    name: "syncdb",
    instance: instanceList[0],
    creatorId: dba.id,
    updaterId: dba.id,
  });

  return instanceList;
};

const createUpdateSchemaStage = (
  server: any,
  workspaceId: string,
  issueId: IssueId,
  creatorId: PrincipalId,
  environmentList: Environment[],
  databaseList: Database[],
  stageStatusList: StageStatus[],
  stepStatusList: StepStatus[]
) => {
  for (let i = 0; i < stageStatusList.length; i++) {
    const stage = server.create("stage", {
      creatorId: creatorId,
      updaterId: creatorId,
      name: `${environmentList[i].name}`,
      type: "bytebase.stage.schema.update",
      status: stageStatusList[i],
      databaseId: databaseList[i].id,
      issueId,
      workspaceId,
    });

    server.create("step", {
      creatorId: creatorId,
      updaterId: creatorId,
      name: "Waiting approval",
      type: "bytebase.step.approve",
      status: stepStatusList[i],
      issueId,
      stage,
      workspaceId,
    });

    server.create("step", {
      creatorId: creatorId,
      updaterId: creatorId,
      name: `Update ${databaseList[i].name} schema`,
      type: "bytebase.step.schema.udpate",
      status: stepStatusList[i],
      issueId,
      stage,
      workspaceId,
    });
  }
};

export default function seeds(server: any) {
  server.loadFixtures();
  workspacesSeeder(server);
}
