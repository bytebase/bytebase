CREATE TABLE principal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creatorId INTEGER NOT NULL,
    createdTs INTEGER NOT NULL,
    updaterId INTEGER NOT NULL,
    updatedTs INTEGER NOT NULL,
    status TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL
);