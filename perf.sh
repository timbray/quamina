go test -run=TestRulerCl2 | grep events/sec | grep -v PREFIX | awk ' {tot += $3; mils = $3 / 1000000.0;  printf("%s: %.3f\n", $1, mils)} END {printf("Avg: %.3f\n", tot / (3.0 * 1000000))}'
