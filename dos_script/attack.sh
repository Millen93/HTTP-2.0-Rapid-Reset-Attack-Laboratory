#!/bin/bash
while true ; do ./rapidresetclient -requests 1000 -url $URL -wait=0 -delay=0 -concurrency=1 ; done 
