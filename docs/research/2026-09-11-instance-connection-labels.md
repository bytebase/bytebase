# Instance connection labels

Research checked 2026-09-11. Recommendation is editorial judgment, not a claim of a universal industry convention.

## Independent reviews

Three independent reviewers researched DBeaver/DataGrip, TablePlus/pgAdmin, and MySQL Workbench/MongoDB Compass. All recommended TLS and SSH tunnel for Bytebase's section labels.

## Verified first-party labels

- DBeaver documents **SSL** and **SSH** configuration tabs. [SSL configuration](https://dbeaver.com/docs/dbeaver/SSL-Configuration/), [SSH configuration](https://dbeaver.com/docs/dbeaver/SSH-Configuration/)
- DataGrip uses an **SSH/SSL** tab with **Use SSL** and **Use SSH tunnel** controls. [DataGrip documentation](https://www.jetbrains.com/help/datagrip/configuring-ssh-and-ssl.html)
- pgAdmin documents an **SSL mode** field, **SSH Tunnel** tab, and **Use SSH tunneling** toggle. [pgAdmin server dialog](https://www.pgadmin.org/docs/pgadmin4/9.17/server_dialog.html)
- TablePlus documents **SSH Tunneling** and describes its `tLSMode` connection parameter as **TLS mode**; these are documentation terms, not verified current form labels. [TablePlus connection documentation](https://docs.tableplus.com/gui-tools/manage-connections)
- MySQL Workbench names its encryption tab **SSL** and its selector **Use SSL**. Its documented values include No, If available, Require, and certificate-verification variants. The short protocol name labels the section while the control indicates behavior. [MySQL Workbench manual](https://dev.mysql.com/doc/workbench/en/wb-mysql-connections-methods-standard.html)
- Workbench calls the SSH connection method **Standard TCP/IP over SSH**. The SSH method still has its own SSL tab: the features compose rather than being alternatives. [MySQL Workbench SSH manual](https://dev.mysql.com/doc/workbench/en/wb-mysql-connections-methods-ssh.html)
- MongoDB Compass documentation explicitly instructs users to select the **TLS / SSL** tab. It documents Default, On, and Off values. The page title is “TLS / SSL Connection Tab,” but the actual tab label omits “Connection.” [Compass TLS documentation](https://www.mongodb.com/docs/compass/connect/advanced-connection-options/tls-ssl-connection/)
- Compass explicitly names its other tab **Proxy / SSH Tunnel**. It documents SSH with Password and SSH with Identity File as options, and supports using SSH tunnels together with TLS. [Compass SSH documentation](https://www.mongodb.com/docs/compass/connect/advanced-connection-options/ssh-connection/)

## Application to Bytebase

Use **TLS** for the visible section and help title, **TLS mode** as the selector accessible label, and **SSH tunnel** for the SSH section, selector accessible label, and help title. Keep TLS choices Disabled / TLS / Mutual TLS and SSH choices Disabled / Enabled.

“TLS options” adds a generic noun already conveyed by the form. “TLS connection” repeats connection context. “TLS mode” is appropriate for the selector but unnecessarily narrows the heading above certificate and identity fields. “SSH tunnel” adds meaningful specificity over bare SSH: it names the network route to the database. “SSH connection” is less specific about the route. Equal word counts are not needed for sibling sections when the extra word conveys a useful distinction.

This is a small label change; no behavioral or layout changes are needed. Existing help text and link labels should be checked so opening help does not switch back to a different section name.
