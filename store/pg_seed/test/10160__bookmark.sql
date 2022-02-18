-- Bookmark for issue 13001 "Hello world"
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
        '/issue/hello-world-13001'
    );

ALTER SEQUENCE bookmark_id_seq RESTART WITH 16002;
