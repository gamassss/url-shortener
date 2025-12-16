#!/bin/bash

API_URL="http://localhost:8080"

for i in $(seq -f "%06g" 1 100); do
    curl -s "$API_URL/hot_$i" > /dev/null &
done
wait

for i in $(seq 1 10000); do
    short_code=$(printf "warm_%06d" $i)
    curl -s "$API_URL/$short_code" > /dev/null &
done
wait