SET search_path TO "second-highest-salary", public;

-- Intentionally left blank: add your solution query locally when solving.
SELECT (SELECT DISTINCT
	salary
FROM
	"Employee" 
ORDER BY
	salary DESC
LIMIT
	1
OFFSET
	1)  AS "SecondHighestSalary";