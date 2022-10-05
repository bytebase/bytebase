#!/bin/bash

# Get parameters
for i in "$@"
do
case $i in
    --files=*)
    FILES="${i#*=}"
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

echo "SQL review start"
echo "Changed files: $FILES"

DOC_URL=https://www.bytebase.com/docs/reference/error-code/advisor
NUM_REGEX='^[0-9]+$'

# junit xml format: https://llg.cubic.org/docs/junit/
xml=""
result=0

for FILE in $FILES; do
    echo "Check file $FILE"

    if ! [[ $FILE =~ \.sql$ ]]; then
        continue
    fi

    echo "Start check statement in file $FILE"

    statement=`cat $FILE`
    if [[ $? != 0 ]]; then
        echo "::error::Cannot open file $FILE"
        exit 1
    fi

    request_body=$(jq -n \
        --arg filePath "$FILE" \
        --arg statement "$statement" \
        '$ARGS.named')
    response=$(curl -s -w "%{http_code}" -X POST $API_URL \
    -H "X-Platform: GitLab" \
    -H "X-Repository: $CI_PROJECT_DIR" \
    -H "X-Actor: $GITLAB_USER_LOGIN" \
    -H "X-Source: action" \
    -H "Content-Type: application/json" \
    -d "$request_body")
    http_code=$(tail -n1 <<< "$response")
    body=$(sed '$ d' <<< "$response")
    echo "::debug::response code: $http_code, response body: $body"

    if [[ $http_code != 200 ]]; then
        echo ":error::Failed to check SQL with response code $http_code and body $body"
        exit 1
    fi

    testcase=""
    index=0

    while read code; do
        content=`echo $body | jq -r ".[$index].content"`
        status=`echo $body | jq -r ".[$index].status"`
        title=`echo $body | jq -r ".[$index].title"`
        line=`echo $body | jq -r ".[$index].line"`
        (( index++ ))

        echo "::debug::status:$status, code:$code, title:$title, line:$line, content:$content"

        if [[ -z "$content" ]]; then
            # The content cannot be empty. Otherwise action cannot output the error message in files.
            content=$title
        fi

        if [[ $code != 0 ]]; then
            title="$title ($code)"

            if ! [[ $line =~ $NUM_REGEX ]] ; then
                line=1
            fi
            if [[ $line -le 0 ]];then
                line=1
            fi

            content="
            File: $FILE
            Line: $line
            Level: $status
            Rule: $title
            Error: $content
            Doc: $DOC_URL#$code"

            testcase="$testcase<testcase name=\"$title\" classname=\"$FILE\" file=\"$FILE#L$line\"><failure>$content</failure></testcase>"

            if [[ $status == 'ERROR' ]]; then
                result=$code
            fi
        fi
    done <<< "$(echo $body | jq -r '.[]' | jq '.code')"

    if [[ ! -z $testcase ]]; then
        xml="$xml<testsuite name=\"$FILE\">$testcase</testsuite>"
    fi
done

if [[ ! -z $xml ]]; then
    echo "output the xml"
    echo $xml
    echo "<?xml version=\"1.0\" encoding=\"UTF-8\"?><testsuites name=\"SQL Review\">$xml</testsuites>" > sql-review.xml
fi

exit $result
