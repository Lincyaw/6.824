#!/bin/bash

cnt=1
# shellcheck disable=SC2034
for i in $(seq 10)
do
   # shellcheck disable=SC2046
    let cnt++
    printf ".."
done