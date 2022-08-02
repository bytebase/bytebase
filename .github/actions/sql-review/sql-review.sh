
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

response=$(curl -s -w "%{http_code}" $API_URL \
  -H "X-Platform: GitHub" \
  -H "X-Repository: $GITHUB_REPOSITORY" \
  -H "X-Actor: $GITHUB_ACTOR" \
  -G --data-urlencode "statement=$statement" \
  -G --data-urlencode "override=$override" \
  -d databaseType=$DATABASE_TYPE \
  -d template=$TEMPLATE_ID)
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
    text="status:$status,code:$code,title:$title,content:$content"
    echo "::debug::$text"

    if [ -z "$content" ]; then
        # The content cannot be empty. Otherwise action cannot output the error message in files.
        content=$title
    fi

    if [ $code != 0 ]; then
        echo "::error::$text"
        title="$title ($code)"
        content="$content
Doc: $DOC_URL#$code"
        content="${content//$'\n'/'%0A'}"
        echo "::error file=$FILE,line=1,col=1,endColumn=2,title=$title::$content"
        result=$code
    fi
done <<< "$(echo $body | jq -r '.[] | "\(.status) \(.code) \(.title) \(.content)"')"

if [ $result != 0 ]; then
    exit 1
fi

exit 0
