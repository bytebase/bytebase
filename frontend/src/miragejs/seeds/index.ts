import faker from "faker";
import { Environment, Stage, Task } from "../../types";
import { instanceSlug, databaseSlug, taskSlug } from "../../utils";

/*
 * Mirage JS guide on Seeds: https://miragejs.com/docs/data-layer/factories#in-development
 */
const workspacesSeeder = (server: any) => {
  // Workspace id is ALWAYS 1 for on-premise deployment
  const workspace1 = server.schema.workspaces.find(1);

  // Workspace 2 is just for verifying we are not returning
  // resources from different workspaces.
  const workspace2 = server.schema.workspaces.find(101);

  // Environment
  let environmentList1 = [];
  for (let i = 0; i < 4; i++) {
    environmentList1.push(
      server.create("environment", {
        workspace: workspace1,
      })
    );
  }
  workspace1.update({ environment: environmentList1 });

  let environmentList2 = [];
  for (let i = 0; i < 4; i++) {
    environmentList2.push(
      server.create("environment", {
        workspace: workspace2,
      })
    );
  }
  workspace2.update({ environment: environmentList2 });

  // Instance
  for (let i = 0; i < 5; i++) {
    server.create("instance", {
      workspace: workspace1,
      name: "instance " + (i + 1) + " " + faker.fake("{{lorem.word}}"),
      // Create an extra instance for prod.
      environmentId: i == 4 ? environmentList1[3].id : environmentList1[i].id,
    });
  }

  for (let i = 0; i < 4; i++) {
    server.create("instance", {
      workspace: workspace2,
      name: "ws2 instance " + (i + 1) + " " + faker.fake("{{lorem.word}}"),
      environmentId: environmentList2[i].id,
    });
  }

  // Task
  const ws1Owner = server.schema.users.find(1);
  const ws1DBA = server.schema.users.find(2);
  const ws1Dev1 = server.schema.users.find(3);
  const ws1Dev2 = server.schema.users.find(5);

  const ws1UserList = [ws1Owner, ws1DBA, ws1Dev1, ws1Dev2];

  const ws2DBA = server.schema.users.find(4);
  const ws2Dev = server.schema.users.find(1);

  let task = server.create("task", {
    type: "bytebase.general",
    name: "Hello, World!",
    description:
      "Welcome to Bytebase, this is the task interface where developers and DBAs collaborate on database management tasks such as: \n\n - Requesting a new database\n - Creating a table\n - Adding a column\n - Troubleshooting performance issue\n\nLet's bookmark this task by clicking the star icon on the top of this page.",
    creatorId: ws1Dev1.id,
    assigneeId: ws1Owner.id,
    subscriberIdList: [ws1DBA.id, ws1Dev2.id],
    stageList: [
      {
        id: "1",
        name: "Request",
        type: "SIMPLE",
        status: "PENDING",
      },
    ],
    workspace: workspace1,
  });

  server.create("activity", {
    actionType: "bytebase.task.create",
    containerId: task.id,
    creator: {
      id: ws1Dev1.id,
      name: ws1Dev1.name,
    },
    workspace: workspace1,
  });

  for (let i = 0; i < 3; i++) {
    const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
    server.create("activity", {
      actionType: "bytebase.task.comment.create",
      containerId: task.id,
      creator: {
        id: user.id,
        name: user.name,
      },
      comment: faker.fake("{{lorem.paragraph}}"),
      workspace: workspace1,
    });
  }

  task = server.create("task", {
    type: "bytebase.database.request",
    name: "Request data source for environment - " + environmentList1[1].name,
    creatorId: ws1Dev1.id,
    assigneeId: ws1Owner.id,
    subscriberIdList: [ws1DBA.id, ws1Dev2.id],
    stageList: [
      {
        id: "1",
        name: "Request data source",
        type: "SIMPLE",
        status: "PENDING",
      },
    ],
    payload: {
      5: environmentList1[1].id,
      7: {
        isNew: true,
        name: "db1",
        readOnly: false,
      },
    },
    workspace: workspace1,
  });

  server.create("activity", {
    actionType: "bytebase.task.create",
    containerId: task.id,
    creator: {
      id: ws1Dev1.id,
      name: ws1Dev1.name,
    },
    workspace: workspace1,
  });

  for (let i = 0; i < 3; i++) {
    const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
    server.create("activity", {
      actionType: "bytebase.task.comment.create",
      containerId: task.id,
      creatorId: user.id,
      comment: faker.fake("{{lorem.paragraph}}"),
      workspace: workspace1,
    });
  }

  const randomUpdateSchemaTaskName = (): string => {
    const list = [
      "Create table " + faker.fake("{{lorem.word}}"),
      "Add index " + faker.fake("{{lorem.word}}"),
      "Drop index " + faker.fake("{{lorem.word}}"),
      "Add column " + faker.fake("{{lorem.word}}"),
      "Drop column " + faker.fake("{{lorem.word}}"),
      "Alter column " + faker.fake("{{lorem.word}}"),
      "Add foreign key " + faker.fake("{{lorem.word}}"),
      "Drop foreign key " + faker.fake("{{lorem.word}}"),
    ];

    return list[Math.floor(Math.random() * list.length)];
  };

  for (let i = 0; i < 5; i++) {
    task = server.create("task", {
      name: randomUpdateSchemaTaskName(),
      type: "bytebase.database.schema.update",
      creatorId: ws1Dev1.id,
      assigneeId: ws1Owner.id,
      creator: {
        id: ws1Dev1.id,
        name: ws1Dev1.name,
      },
      subscriberIdList: [ws1DBA.id, ws1Dev2.id],
      ...fillTaskAndStageStatus(environmentList1),
      workspace: workspace1,
    });

    server.create("activity", {
      actionType: "bytebase.task.create",
      containerId: task.id,
      creatorId: ws1Dev1.id,
      workspace: workspace1,
    });

    for (let i = 0; i < 3; i++) {
      const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
      server.create("activity", {
        actionType: "bytebase.task.comment.create",
        containerId: task.id,
        creatorId: user.id,
        comment: faker.fake("{{lorem.paragraph}}"),
        workspace: workspace1,
      });
    }
  }

  for (let i = 0; i < 15; i++) {
    task = server.create("task", {
      name: randomUpdateSchemaTaskName(),
      type: "bytebase.database.schema.update",
      creatorId: ws1Owner.id,
      assigneeId: ws1DBA.id,
      subscriberIdList: [ws1Dev2.id],
      ...fillTaskAndStageStatus(environmentList1),
      workspace: workspace1,
    });

    server.create("activity", {
      actionType: "bytebase.task.create",
      containerId: task.id,
      creator: {
        id: ws1Owner.id,
        name: ws1Owner.name,
      },
      workspace: workspace1,
    });

    for (let i = 0; i < 3; i++) {
      const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
      server.create("activity", {
        actionType: "bytebase.task.comment.create",
        containerId: task.id,
        creatorId: user.id,
        comment: faker.fake("{{lorem.paragraph}}"),
        workspace: workspace1,
      });
    }
  }

  for (let i = 0; i < 15; i++) {
    task = server.create("task", {
      name: randomUpdateSchemaTaskName(),
      type: "bytebase.database.schema.update",
      creatorId: ws1Dev2.id,
      assigneeId: ws1DBA.id,
      subscriberIdList: [ws1Owner.id, ws1Dev1.id],
      ...fillTaskAndStageStatus(environmentList1),
      workspace: workspace1,
    });

    server.create("activity", {
      actionType: "bytebase.task.create",
      containerId: task.id,
      creatorId: ws1Dev2.id,
      workspace: workspace1,
    });

    for (let i = 0; i < 3; i++) {
      const user = ws1UserList[Math.floor(Math.random() * ws1UserList.length)];
      server.create("activity", {
        actionType: "bytebase.task.comment.create",
        containerId: task.id,
        creatorId: user.id,
        comment: faker.fake("{{lorem.paragraph}}"),
        workspace: workspace1,
      });
    }
  }

  task = server.create("task", {
    name: randomUpdateSchemaTaskName(),
    type: "bytebase.database.schema.update",
    creatorId: ws2Dev.id,
    assigneeId: ws2DBA.id,
    ...fillTaskAndStageStatus(environmentList2),
    workspace: workspace2,
  });

  server.create("activity", {
    actionType: "bytebase.task.create",
    containerId: task.id,
    creatorId: ws1Dev1.id,
    workspace: workspace2,
  });

  // Workspace 2
  // Task 3
  const task3 = server.schema.tasks.findBy({
    workspaceId: workspace2.id,
  });
  server.create("bookmark", {
    workspace: workspace2,
    name: task3.name,
    link: `/task/${taskSlug(task3.name, task3.id)}`,
    creatorId: ws1Owner.id,
  });
};

const fillTaskAndStageStatus = (
  environmentList: Environment[]
): Pick<Task, "status" | "stageList"> => {
  const i = Math.floor(Math.random() * 5);
  if (i % 5 == 0) {
    return {
      status: "OPEN",
      stageList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
      ],
    };
  } else if (i % 5 == 1) {
    return {
      status: "OPEN",
      stageList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "RUNNING",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
      ],
    };
  } else if (i % 5 == 2) {
    return {
      status: "DONE",
      stageList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "SKIPPED",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
      ],
    };
  } else if (i % 5 == 3) {
    return {
      status: "OPEN",
      stageList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "FAILED",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
      ],
    };
  } else {
    return {
      status: "CANCELED",
      stageList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "SKIPPED",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
      ],
    };
  }
};

export default function seeds(server: any) {
  server.loadFixtures();
  workspacesSeeder(server);
}
