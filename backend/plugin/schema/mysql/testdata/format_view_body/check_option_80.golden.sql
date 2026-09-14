select
  `t1`.`a` AS `a`,
  `t1`.`c` AS `c`
from `t1`
where (`t1`.`c` > 5)
WITH CASCADED CHECK OPTION
