#!/bin/bash

function slurm_change_node_state(){
  for node in $(sinfo -N --Format=NodeList,StateComplete -h | awk '$2 ~ /fail|down|drain/ { print $1 }'); do
     echo "$node execute command 'scontrol update nodename=$node state=idle'"
     /usr/bin/scontrol update nodename=$node state=resume
  done
}

function slurm_reconfigure(){
  /usr/bin/scontrol reconfigure
}

# slurm recovering
slurm_reconfigure
slurm_change_node_state

# close munged
munge_pid=$(ps -ef | grep 'munge'| awk '$8=="/usr/sbin/munged" {print $2}')
echo "munge pid: $munge_pid"
kill -9 $munge_pid
