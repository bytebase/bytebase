-- Project 3001 hook
INSERT INTO
    project_webhook (
        id,
        creator_id,
        updater_id,
        project_id,
        type,
        name,
        url,
        activity_list
    )
VALUES
    (
        4101,
        101,
        101,
        3001,
        'bb.plugin.webhook.wecom',
        'WeCom',
        'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=12345',
        '{"bb.issue.create", "bb.issue.status.update", "bb.pipeline.task.status.update", "bb.issue.field.update", "bb.issue.comment.create"}'
    );

INSERT INTO
    project_webhook (
        id,
        creator_id,
        updater_id,
        project_id,
        type,
        name,
        url,
        activity_list
    )
VALUES
    (
        4102,
        101,
        101,
        3001,
        'bb.plugin.webhook.dingtalk',
        'DingTalk',
        'https://oapi.dingtalk.com/robot/send?access_token=12345',
        '{"bb.issue.create", "bb.issue.status.update", "bb.pipeline.task.status.update", "bb.issue.field.update", "bb.issue.comment.create"}'
    );

-- Project 3002 hook
INSERT INTO
    project_webhook (
        id,
        creator_id,
        updater_id,
        project_id,
        type,
        name,
        url,
        activity_list
    )
VALUES
    (
        4103,
        101,
        101,
        3002,
        'bb.plugin.webhook.feishu',
        'Feishu',
        'https://open.feishu.cn/open-apis/bot/v2/hook/12345',
        '{"bb.issue.create", "bb.issue.status.update", "bb.pipeline.task.status.update", "bb.issue.field.update", "bb.issue.comment.create"}'
    );

INSERT INTO
    project_webhook (
        id,
        creator_id,
        updater_id,
        project_id,
        type,
        name,
        url,
        activity_list
    )
VALUES
    (
        4104,
        101,
        101,
        3002,
        'bb.plugin.webhook.slack',
        'Slack',
        'https://hooks.slack.com/services/12345',
        '{"bb.issue.create", "bb.issue.status.update", "bb.pipeline.task.status.update", "bb.issue.field.update", "bb.issue.comment.create"}'
    );

-- Project 3003 hook
INSERT INTO
    project_webhook (
        id,
        creator_id,
        updater_id,
        project_id,
        type,
        name,
        url,
        activity_list
    )
VALUES
    (
        4105,
        101,
        101,
        3003,
        'bb.plugin.webhook.discord',
        'Discord',
        'https://discord.com/api/webhooks/12345',
        '{"bb.issue.create", "bb.issue.status.update", "bb.pipeline.task.status.update", "bb.issue.field.update", "bb.issue.comment.create"}'
    );

INSERT INTO
    project_webhook (
        id,
        creator_id,
        updater_id,
        project_id,
        type,
        name,
        url,
        activity_list
    )
VALUES
    (
        4106,
        101,
        101,
        3003,
        'bb.plugin.webhook.teams',
        'Teams',
        'https://foo.webhook.office.com/webhookb2/12345',
        '{"bb.issue.create", "bb.issue.status.update", "bb.pipeline.task.status.update", "bb.issue.field.update", "bb.issue.comment.create"}'
    );

ALTER SEQUENCE project_webhook_id_seq RESTART WITH 4107;
