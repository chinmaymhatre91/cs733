#!/bin/bash

pids=$(ps -A | grep "$1" | grep -v grep | grep -v "kill"| awk '{print $1}')
for i in $pids
do
	kill -9 $i
done
#echo "killed "$1