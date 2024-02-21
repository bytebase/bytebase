#!/bin/bash
# ===========================================================================
# File: bytebase-sql-review.sh
# Description: usage: ./bytebase-sql-review.sh --pull-request-id=[pr id] --repo-id=[repo id] --repo-url=[repo URL] --api=[API URL] --token=[API token]
# ===========================================================================

# Get parameters
for i in "$@"
do
case $i in
    --pull-request-id=*)
    PR_ID="${i#*=}"
    shift
    ;;
    --repo-id=*)
    REPO_ID="${i#*=}"
    shift
    ;;
    --repo-url=*)
    REPO_URL="${i#*=}"
    shift
    ;;
    --token=*)
    API_TOKEN="${i#*=}"
    shift
    ;;
    --api=*)
    API_URL="${i#*=}"
    shift
    ;;
    *) # unknown option
    ;;
esac
done

echo "Start request $API_URL"

if [[ $PR_ID == "" ]]; then exit 0; fi

request_body=$(jq -n \
--arg 'repositoryId' "$REPO_ID" \
--arg 'pullRequestId' "$PR_ID" \
--arg 'webURL' "$REPO_URL" \
'$ARGS.named')

response=$(curl -s --show-error -X POST $API_URL \
-H "X-SQL-Review-Token: $API_TOKEN" \
-H "Content-Type: application/json" \
-d "$request_body")

echo "response: $response"

content=$(echo "$response" | jq -r '.content')
status=$(echo "$response" | jq -r '.status')

len=$(echo "$content" | jq '. | length')
if [[ $len == 0 ]]; then exit 0; fi

msg=$(echo "$content" | jq -r '.[0]')
mkdir test-results
echo $msg >> test-results/bytebase-sql-review.xml

if [ "$status" != "SUCCESS" ]; then exit 1; fi
