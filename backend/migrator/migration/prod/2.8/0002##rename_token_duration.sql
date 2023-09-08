UPDATE setting
SET value = REPLACE(value, '"refreshTokenDuration":', '"tokenDuration":')
WHERE name = 'bb.workspace.profile';
