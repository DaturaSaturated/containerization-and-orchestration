#!/bin/bash
while true; do
  T=$(date +%s.%N)
  CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 http://127.0.0.1:30080/health)
  echo "$T,$CODE" >> load.csv
  sleep 0.5
done
