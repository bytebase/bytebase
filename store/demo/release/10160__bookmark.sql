-- Bookmark for issue 101 "Hello world"
INSERT INTO
    bookmark (
        id,
        creator_id,
        updater_id,
        name,
        link
    )
VALUES
    (
        16001,
        101,
        101,
        'Hello world!',
        '/issue/hello-world-101'
    );

ALTER SEQUENCE bookmark_id_seq RESTART WITH 16002;
