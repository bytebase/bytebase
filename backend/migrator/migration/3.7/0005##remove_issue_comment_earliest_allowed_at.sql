DELETE FROM issue_comment WHERE (payload->'taskUpdate')?|'{toEarliestAllowedTime, fromEarliestAllowedTime}'; 
