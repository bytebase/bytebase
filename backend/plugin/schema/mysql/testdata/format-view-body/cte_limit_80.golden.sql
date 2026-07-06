with `x` as (select `t1`.`a` AS `a`,`t1`.`b` AS `b` from `t1` where (`t1`.`c` > 0))
select
  `x`.`a` AS `a`,
  `x`.`b` AS `b`
from `x`
order by `x`.`a`
limit 10
