#!/bin/bash
source ../aws_test_config.sh
go run ../cmd/master/server/main.go ../cmd/master/server/server.go &
SERVER_PID=$!
sleep 5

echo "This is a test file for EchoFS AWS integration." > test_file.txt

curl -s http://localhost:8080/api/v1/health | jq '.'
curl -X POST http://localhost:8080/api/v1/files/upload \
  -F "file=@test_file.txt" \
  -F "user_id=test-user" \
  -s | jq '.'
curl -s http://localhost:8080/api/v1/files | jq '.data | length'

rm -f test_file.txt
kill $SERVER_PID 2>/dev/null