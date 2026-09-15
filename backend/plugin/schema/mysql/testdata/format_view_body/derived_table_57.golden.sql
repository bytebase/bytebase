select
  `dt`.`s` AS `s`
from (select sum(`t1`.`c`) AS `s` from `t1` group by `t1`.`b`) `dt`
where (`dt`.`s` > 1)
