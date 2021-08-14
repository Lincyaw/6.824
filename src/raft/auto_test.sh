#!/bin/bash
read -r -p "please input test function: " lab
read -r -p "please input iteration number: " inter

# shellcheck disable=SC2034
for i in $(seq "$inter")
do
   # shellcheck disable=SC2094
   go test -race -run "$lab" >> "$lab".log
done
