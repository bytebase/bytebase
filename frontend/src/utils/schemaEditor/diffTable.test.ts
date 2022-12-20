import { expect, it } from "vitest";
import {
  DropTableContext,
  CreateTableContext,
  AlterTableContext,
  RenameTableContext,
} from "@/types";
import { Table } from "@/types/schemaEditor/atomType";
import { UNKNOWN_ID } from "@/types/const";
import { diffTableList } from "./diffTable";

it("diff create table list", () => {
  const testList: {
    originTableList: Table[];
    targetTableList: Table[];
    wanted: {
      createTableList: CreateTableContext[];
      alterTableList: AlterTableContext[];
      renameTableList: RenameTableContext[];
      dropTableList: DropTableContext[];
    };
  }[] = [
    {
      originTableList: [],
      targetTableList: [
        {
          oldName: "user",
          newName: "user",
          engine: "InnoDB",
          collation: "",
          comment: "",
          originColumnList: [],
          columnList: [
            {
              id: UNKNOWN_ID,
              oldName: "id",
              newName: "id",
              type: "int",
              comment: "",
              nullable: false,
              status: "created",
            },
          ],
          status: "created",
        } as any as Table,
      ],
      wanted: {
        createTableList: [
          {
            name: "user",
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
        renameTableList: [],
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
      renameTableList: RenameTableContext[];
      dropTableList: DropTableContext[];
    };
  }[] = [
    {
      originTableList: [
        {
          oldName: "user",
          newName: "user",
          engine: "InnoDB",
          collation: "",
          comment: "",
          columnList: [
            {
              oldName: "id",
              newName: "id",
              type: "int",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              status: "normal",
            },
          ],
          status: "normal",
        } as any as Table,
      ],
      targetTableList: [
        {
          oldName: "user",
          newName: "user",
          engine: "InnoDB",
          collation: "",
          comment: "",
          columnList: [
            {
              oldName: "id",
              newName: "id",
              type: "int",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              status: "normal",
            },
            {
              oldName: "email",
              newName: "email",
              type: "varchar",
              characterSet: "",
              collation: "",
              comment: "",
              nullable: false,
              status: "created",
            },
          ],
          status: "normal",
        } as any as Table,
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
                default: undefined,
                nullable: false,
              },
            ],
            changeColumnList: [],
            dropColumnList: [],
          },
        ],
        renameTableList: [],
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
      renameTableList: RenameTableContext[];
      dropTableList: DropTableContext[];
    };
  }[] = [
    {
      originTableList: [
        {
          id: 1,
          oldName: "user",
          newName: "user",
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
            },
          ],
          status: "normal",
        } as any as Table,
      ],
      targetTableList: [
        {
          id: 1,
          oldName: "user",
          newName: "user",
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
            },
          ],
          status: "dropped",
        } as any as Table,
      ],
      wanted: {
        createTableList: [],
        alterTableList: [],
        renameTableList: [],
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
