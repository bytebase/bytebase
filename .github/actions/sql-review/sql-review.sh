
FILE=$1
DATABASE_TYPE=$2
CONFIG=$3
TEMPLATE_ID=$4

# Users can set their own SQL review API URL into the environment variable "BB_SQL_API"
API_URL=$BB_SQL_API
if [ -z $API_URL ]
then
    # TODO: replace the url
    API_URL=https://sql-service.onrender.com/v1/sql/advise
fi
DOC_URL=https://www.bytebase.com/docs/reference/error-code/advisor

statement=`cat $FILE`
if [ $? != 0 ]
then
    echo "::error::Cannot open file $FILE"
    exit 1
fi

override=""
if [ ! -z $CONFIG ]
then
    override=`cat $CONFIG`

    if [ $? != 0 ]
    then
        echo "::error::Cannot find SQL review config file"
        exit 1
    fi
fi

request_body=$(jq -n \
    --arg statement "$statement" \
    --arg override "$override" \
    --arg databaseType "$DATABASE_TYPE" \
    --arg templateId "$TEMPLATE_ID" \
    '$ARGS.named')
response=$(curl -s -w "%{http_code}" -X POST $API_URL \
  -H "X-Platform: GitHub" \
  -H "X-Repository: $GITHUB_REPOSITORY" \
  -H "X-Actor: $GITHUB_ACTOR" \
  -H "Content-Type: application/json" \
  -d "$request_body")

http_code=$(tail -n1 <<< "$response")
body=$(sed '$ d' <<< "$response")

echo "::debug::response code: $http_code, response body: $body"

if [ $http_code != 200 ]
then
    echo "::error::failed to check SQL with response code $http_code and body $body"
    exit 1
fi

result=0
while read status code title content; do
    echo "::debug::status:$status,code:$code,title:$title,content:$content"

    if [ -z "$content" ]; then
        # The content cannot be empty. Otherwise action cannot output the error message in files.
        content=$title
    fi

    if [ $code != 0 ]; then
        title="$title ($code)"
        content="$content
Doc: $DOC_URL#$code"
        content="${content//$'\n'/'%0A'}"
        error_msg="file=$FILE,line=1,col=1,endColumn=2,title=$title::$content"

        if [ $status == 'WARN' ]; then
            echo "::warning $error_msg"
        else
            echo "::error $error_msg"
            result=$code
        fi
    fi
done <<< "$(echo $body | jq -r '.[] | "\(.status) \(.code) \(.title) \(.content)"')"

exit $result
