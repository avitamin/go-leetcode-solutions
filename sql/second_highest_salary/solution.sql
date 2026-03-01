SET search_path TO "second-highest-salary", public;

-- Intentionally left blank: add your solution query locally when solving.
select DISTINCT * from "second-highest-salary"."Employee" limit 1 OFFSET 1;