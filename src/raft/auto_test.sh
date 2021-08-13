#!/bin/bash
read -r -p "please input test number: " lab
read -r -p "please input iteration number: " inter

# shellcheck disable=SC2034
for i in $(seq "$inter")
do
   go test -race -run "$lab" >> logs
done
