select
  concat(`t1`.`b`,' from ','x\'y','z"q') AS `weird`
from `t1`
where (`t1`.`b` <> 'union all')
