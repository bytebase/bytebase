This folder contains the seed data. Each folder contains seed files for the corresponding mode

Each file represents the data for a particular table. The file name is something like 00XX, reasons for doing this:

- Code loads the seed data in lexical file order, thus we maintain an explict order about how seed data is loaded since there exists order dependency between tables.

- For dev mode, XX also corresponds to the id prefix for the record, so we can identify the resource type by just looking the id, and catch errors that our code accidentally passes an ID for a different resource type.
