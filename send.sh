#!/bin/bash

TODAY=$(date "+%Y%m%d")
DIR="."
AUTH_FILE="${DIR}/auth-detail-${TODAY}"
DETAIL_FILE="${DIR}/detail-${TODAY}"

declare -p DETAIL_FILE

#DIR="/var/log/freeradius/radacct/10.228.76.154"
CAP_URL="https://localhost:9090"
ENDPOINT="/api/post_context"

# SAVED_FILE_PATH="/home/hirai/send.tmp"
SAVED_FILE_PATH="./send.tmp"

if [ -e ${SAVED_FILE_PATH} ]; then
	mapfile -t current_state < ${SAVED_FILE_PATH}
else
	declare -a current_state
fi
declare -p current_state


if [[ ${current_state[0]} != ${TODAY} ]]; then
	current_state[0]=${TODAY}
	current_state[1]=0
	current_state[2]=0
fi

if [ -e ${AUTH_FILE} ]; then
	auth_payload=$(cat "${AUTH_FILE}" | awk "NR>=${current_state[1]} {print}")
	current_state[1]=$(($(cat "${AUTH_FILE}" | wc -l)+1))
fi
if [ -e ${DETAIL_FILE} ]; then
	detail_payload=$(cat "${DETAIL_FILE}" | awk "NR>=${current_state[2]} {print}")
	current_state[2]=$(($(cat "${DETAIL_FILE}" | wc -l)+1))
fi


if [[ ${auth_payload}${detail_payload} != "" ]]; then
	curl -XPOST ${CAP_URL}${ENDPOINT} --data-urlencode "auth=${auth_payload}" --data-urlencode "detail=${detail_payload}" \
	--cacert server.crt
fi

if [[ $? -eq 0 ]]; then
	IFS=$'\n'
	echo "${current_state[*]}" >| ${SAVED_FILE_PATH}
	unset IFS
	declare -p current_state
fi