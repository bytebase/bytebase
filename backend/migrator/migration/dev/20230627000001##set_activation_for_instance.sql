UPDATE instance
SET activation = TRUE
FROM (SELECT value FROM setting WHERE name = 'bb.enterprise.license') AS license
WHERE license.value != '';