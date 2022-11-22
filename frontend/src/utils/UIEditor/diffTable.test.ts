import { expect, it } from "vitest";
import {
  Table,
  DropTableContext,
  CreateTableContext,
  AlterTableContext,
} from "@/types";
import { UNKNOWN_ID } from "@/types/const";
import { diffTableList } from "./diffTable";

it("diff create table list", () => {
  const testList: {
    originTableList: Table[];
    targetTableList: Table[];
    wanted: {
      createTableList: CreateTableContext[];
      alterTableList: AlterTableContext[];
      dropTableList: DropTableContext[];
    };
  }[] = [
    {
      originTableList: [],
      targetTableList: [
        {
          name: "user",
          type: "BASE TABLE",
          engine: "InnoDB",
          collation: "",
          comment: "",
          columnList: [
            {
              id: UNKNOWN_ID,
              name: "id",
              type: "int",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              default: undefined,
            },
          ],
        } as Table,
      ],
      wanted: {
        createTableList: [
          {
            name: "user",
            type: "BASE TABLE",
            engine: "InnoDB",
            characterSet: "",
            collation: "",
            comment: "",
            addColumnList: [
              {
                name: "id",
                type: "int",
                characterSet: "",
                collation: "",
                comment: "",
                nullable: false,
                default: undefined,
              },
            ],
          },
        ],
        alterTableList: [],
        dropTableList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffTableList(test.originTableList, test.targetTableList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff alter table list", () => {
  const testList: {
    originTableList: Table[];
    targetTableList: Table[];
    wanted: {
      createTableList: CreateTableContext[];
      alterTableList: AlterTableContext[];
      dropTableList: DropTableContext[];
    };
  }[] = [
    {
      originTableList: [
        {
          name: "user",
          type: "BASE TABLE",
          engine: "InnoDB",
          collation: "",
          comment: "",
          columnList: [
            {
              name: "id",
              type: "int",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              default: undefined,
            },
          ],
        } as Table,
      ],
      targetTableList: [
        {
          name: "user",
          type: "BASE TABLE",
          engine: "InnoDB",
          collation: "",
          comment: "",
          columnList: [
            {
              name: "id",
              type: "int",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              default: undefined,
            },
            {
              id: UNKNOWN_ID,
              name: "email",
              type: "varchar",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              default: undefined,
            },
          ],
        } as Table,
      ],
      wanted: {
        createTableList: [],
        alterTableList: [
          {
            name: "user",
            addColumnList: [
              {
                name: "email",
                type: "varchar",
                characterSet: "",
                collation: "",
                comment: "",
                nullable: false,
                default: undefined,
              },
            ],
            modifyColumnList: [],
            dropColumnList: [],
          },
        ],
        dropTableList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffTableList(test.originTableList, test.targetTableList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff drop table list", () => {
  const testList: {
    originTableList: Table[];
    targetTableList: Table[];
    wanted: {
      createTableList: CreateTableContext[];
      alterTableList: AlterTableContext[];
      dropTableList: DropTableContext[];
    };
  }[] = [
    {
      originTableList: [
        {
          name: "user",
          type: "BASE TABLE",
          engine: "InnoDB",
          collation: "",
          comment: "",
          columnList: [
            {
              name: "id",
              type: "int",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              default: undefined,
            },
          ],
        } as Table,
      ],
      targetTableList: [],
      wanted: {
        createTableList: [],
        alterTableList: [],
        dropTableList: [
          {
            name: "user",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffTableList(test.originTableList, test.targetTableList);
    expect(result).toStrictEqual(test.wanted);
  }
});
