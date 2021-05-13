This folder contains the seed data used for testing

Each file represents the testing data for a particular table. The file name is something like 00XX where XX also corresponds to the id prefix for the record. This has couple benefits:

- We can identify the resource type by just looking the id.
- Catch errors that our code accidentally passes an ID for a different resource type.
