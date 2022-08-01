
FILE=$1
DATABASE_TYPE=$2
CONFIG=$3

statement=`cat $FILE`
if [ $? != 0 ]
then
    exit 1
fi

override=`cat $CONFIG`
if [ $? != 0 ]
then
    exit 1
fi


# TODO: replace the url
response=$(curl -s -w "%{http_code}" https://sql-service.onrender.com/v1/sql/advise \
  -G --data-urlencode "statement=$statement" \
  -G --data-urlencode "override=$override" \
  -d databaseType=$DATABASE_TYPE)
http_code=$(tail -n1 <<< "$response")
body=$(sed '$ d' <<< "$response")

echo "::debug::response code: $http_code, response body: $body"

if [ $http_code != 200 ]
then
    echo "failed to check SQL with response code $http_code and body $body"
    exit 1
fi

result=0
while read i; do
    echo "::debug::$i"
    status=$(jq -r '.status' <<< "$i")
    code=$(jq -r '.code' <<< "$i")
    title=$(jq -r '.title' <<< "$i")
    content=$(jq -r '.content' <<< "$i")

    if [ -z "$content" ]; then
        content=$title
    fi

    if [ $code != 0 ]; then
        echo $i
        echo "::error file=$FILE,line=1,col=1,endColumn=2,title=$title::$content"
        result=$code
    fi
done <<< "$(echo $body | jq -c '.[]')"

if [ $result != 0 ]; then
    exit 1
fi

exit 0
