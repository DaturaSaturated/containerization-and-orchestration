#!/bin/bash
while true; do
  POD=$(kubectl get pods -n default -l 'app in (shop-api,shop-worker)' --no-headers | shuf -n1 | awk '{print $1}')
  [ -n "$POD" ] && echo "$(date +%s.%N) KILL $POD" >> chaos.log && kubectl delete pod -n default "$POD" --wait=false
  sleep 5
done
