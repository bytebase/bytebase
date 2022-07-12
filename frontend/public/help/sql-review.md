---
title: SQL Review
---

SQL Review is the classic and widely used process for developers and DBAs to coordinate database schema changes.

Developers first submit their SQL statement for DBA to review. After review is approved, the SQL statement will then be applied to the corresponding database. For a single change, this step would normally be repeated for each environment (e.g. integration, staging, prod).

To facilitate this heavily used process, Bytebase has designed a comprehensive interface to unify the stage progression and the collaboration experience into a single page.

![Issue detail interface](https://bytebase.com/static/docs/issue-view-annotated.png)

1. Status banner highlighting the issue status.

2. Issue abstract. If the issue is created by a code commit from a linked repository, that code commit information will be displayed.

3. Action buttons to change the issue status.

4. Flow bar showing the stage progression.

5. Task execution status for the selected stage. By default, the current active stage is selected. User can click the flow bar (Item 4) or use the stage dropdown on the right side bar (Item 10) to switch the stage.

6. SQL statement for the selected stage. The statement could also change by selecting different stages.

7. Issue description.

8. Issue activity and comment area.

9. Issue status and assignee.

10. Stage info. User can use the stage dropdown to check info from other stages.

11. Issue misc info.

12. Subscription list.
