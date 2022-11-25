import { expect, it } from "vitest";
import {
  AddColumnContext,
  Column,
  DropColumnContext,
  ChangeColumnContext,
} from "@/types";
import { UNKNOWN_ID } from "@/types/const";
import { diffColumnList } from "./diffColumn";

it("diff add column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [],
      targetColumnList: [
        {
          id: UNKNOWN_ID,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: undefined,
        } as Column,
      ],
      wanted: {
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
        changeColumnList: [],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff modify column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          id: 1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [
        {
          id: 1,
          name: "id",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: undefined,
        } as Column,
      ],
      wanted: {
        addColumnList: [],
        changeColumnList: [
          {
            oldName: "id",
            newName: "id",
            type: "varchar",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            default: undefined,
          },
        ],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff drop column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          id: 1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [],
      wanted: {
        addColumnList: [],
        changeColumnList: [],
        dropColumnList: [
          {
            name: "id",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          id: 1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
        {
          id: 2,
          name: "name",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
        {
          id: 3,
          name: "city",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [
        {
          id: 1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "this is id",
          nullable: true,
          default: undefined,
        } as Column,
        {
          id: 2,
          name: "name",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: "",
        } as Column,
        {
          id: UNKNOWN_ID,
          name: "birthday",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: "",
        } as Column,
      ],
      wanted: {
        addColumnList: [
          {
            name: "birthday",
            type: "varchar",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            default: "",
          },
        ],
        changeColumnList: [
          {
            oldName: "id",
            newName: "id",
            type: "int",
            characterSet: "",
            collation: "",
            comment: "this is id",
            nullable: true,
            default: undefined,
          },
          {
            oldName: "name",
            newName: "name",
            type: "varchar",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            default: "",
          },
        ],
        dropColumnList: [
          {
            name: "city",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});
