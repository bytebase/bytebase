This folder contains the seed data for test. The test seed data is used when NOT building with "release" tag or running with --demo. Test seed data is
reloaded everytime we start the application (by setting forceResetSeed in the dev profile from the code)

Each file represents the data for a particular table. The file name is something like YXXXX, where Y is the major version number and XXXX is the id
start sequence number for that table, reasons for doing this:

- Y (majar version number) corresponds to the major schema version the seed file applies to

- Code loads the seed data in lexical file order, thus we maintain an explict order about how seed data is loaded since there exists order dependency between tables.

- Because XXXX corresponds to the id prefix for the record, so we can identify the resource type by just looking the id, and catch errors that our code accidentally passes an ID for a different resource type.
