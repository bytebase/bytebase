select
  `a`.`actor_id` AS `actor_id`,
  `a`.`first_name` AS `first_name`,
  `a`.`last_name` AS `last_name`,
  group_concat(distinct concat(`c`.`name`,': ',(select group_concat(`f`.`title` order by `f`.`title` ASC separator ', ') from ((`film` `f` join `film_category` `fc` on((`f`.`film_id` = `fc`.`film_id`))) join `film_actor` `fa` on((`f`.`film_id` = `fa`.`film_id`))) where ((`fc`.`category_id` = `c`.`category_id`) and (`fa`.`actor_id` = `a`.`actor_id`)))) order by `c`.`name` ASC separator '; ') AS `film_info`
from (((`actor` `a`
  left join `film_actor` `fa` on((`a`.`actor_id` = `fa`.`actor_id`)))
  left join `film_category` `fc` on((`fa`.`film_id` = `fc`.`film_id`)))
  left join `category` `c` on((`fc`.`category_id` = `c`.`category_id`)))
group by `a`.`actor_id`,`a`.`first_name`,`a`.`last_name`
