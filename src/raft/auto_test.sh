#!/bin/bash
read -r -p "please input test function: " lab
read -r -p "please input iteration number: " inter

folder="test_logs"

if [ -d $folder ];then
  echo "Entering "$folder...
else
  mkdir $folder
fi

if [ -f $folder/"$lab" ];then
    rm $folder/"$lab".log
fi
echo "Test begin... Please wait~"

tmplog=/tmp/tmplog
# shellcheck disable=SC2034
for i in $(seq "$inter")
do
   # shellcheck disable=SC2094
   go test -race -run "$lab" > $tmplog
   # shellcheck disable=SC2046
   if [ $(grep -c "FAIL" $tmplog) -ne '0' ]; then
      cat $tmplog >> $folder/"$lab".log
      printf "\n----- ----- ----- ----- -----\n\n" >> $folder/"$lab".log
   fi
done
rm $tmplog

if [ -f $folder/"$lab" ];then
    failnumber=$(grep -c "FAIL" $folder/"$lab".log)
    if [ "$failnumber" -ne "0" ];then
      echo "Test ends, found $failnumber failures. See log file in $(pwd)/$folder/$lab.log"
    else
      echo "Test ends, no failure! Congratulations!! "
    fi
else
    echo "Test ends, no failure! Congratulations!! "
fi


