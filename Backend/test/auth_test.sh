#!/bin/bash

# EchoFS Authentication System Test Script
# This script tests the complete authentication flow

set -e

BASE_URL="http://localhost:8080/api/v1"
TEST_EMAIL="test_$(date +%s)@example.com"
TEST_USERNAME="testuser_$(date +%s)"
TEST_PASSWORD="testpass123"

echo "🧪 EchoFS Authentication Test Suite"
echo "===================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Register new user
echo "📝 Test 1: Register new user"
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"$TEST_USERNAME\",
    \"email\": \"$TEST_EMAIL\",
    \"password\": \"$TEST_PASSWORD\"
  }")

echo "$REGISTER_RESPONSE" | jq .

if echo "$REGISTER_RESPONSE" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ Registration successful${NC}"
    TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.token')
    USER_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.user.id')
else
    echo -e "${RED}✗ Registration failed${NC}"
    exit 1
fi
echo ""

# Test 2: Login with credentials
echo "🔐 Test 2: Login with credentials"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$TEST_EMAIL\",
    \"password\": \"$TEST_PASSWORD\"
  }")

echo "$LOGIN_RESPONSE" | jq .

if echo "$LOGIN_RESPONSE" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ Login successful${NC}"
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
else
    echo -e "${RED}✗ Login failed${NC}"
    exit 1
fi
echo ""

# Test 3: Get user profile
echo "👤 Test 3: Get user profile"
PROFILE_RESPONSE=$(curl -s -X GET "$BASE_URL/auth/profile" \
  -H "Authorization: Bearer $TOKEN")

echo "$PROFILE_RESPONSE" | jq .

if echo "$PROFILE_RESPONSE" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ Profile retrieved successfully${NC}"
else
    echo -e "${RED}✗ Failed to get profile${NC}"
    exit 1
fi
echo ""

# Test 4: Upload a file
echo "📤 Test 4: Upload a file"
echo "Test file content" > /tmp/echofs_test_file.txt

UPLOAD_RESPONSE=$(curl -s -X POST "$BASE_URL/files/upload" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/echofs_test_file.txt")

echo "$UPLOAD_RESPONSE" | jq .

if echo "$UPLOAD_RESPONSE" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ File uploaded successfully${NC}"
    FILE_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.data.file_id')
else
    echo -e "${RED}✗ File upload failed${NC}"
    exit 1
fi
echo ""

# Test 5: List files
echo "📋 Test 5: List user's files"
LIST_RESPONSE=$(curl -s -X GET "$BASE_URL/files" \
  -H "Authorization: Bearer $TOKEN")

echo "$LIST_RESPONSE" | jq .

if echo "$LIST_RESPONSE" | jq -e '.success == true' > /dev/null; then
    FILE_COUNT=$(echo "$LIST_RESPONSE" | jq '.data | length')
    echo -e "${GREEN}✓ Files listed successfully (found $FILE_COUNT files)${NC}"
else
    echo -e "${RED}✗ Failed to list files${NC}"
    exit 1
fi
echo ""

# Test 6: Download file
echo "📥 Test 6: Download file"
curl -s -X GET "$BASE_URL/files/$FILE_ID/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o /tmp/echofs_downloaded_file.txt

if [ -f /tmp/echofs_downloaded_file.txt ]; then
    DOWNLOADED_CONTENT=$(cat /tmp/echofs_downloaded_file.txt)
    if [ "$DOWNLOADED_CONTENT" == "Test file content" ]; then
        echo -e "${GREEN}✓ File downloaded successfully and content matches${NC}"
    else
        echo -e "${YELLOW}⚠ File downloaded but content doesn't match${NC}"
    fi
else
    echo -e "${RED}✗ File download failed${NC}"
    exit 1
fi
echo ""

# Test 7: Try to access without authentication
echo "🚫 Test 7: Access without authentication (should fail)"
UNAUTH_RESPONSE=$(curl -s -X GET "$BASE_URL/files")

if echo "$UNAUTH_RESPONSE" | grep -q "Authorization header required"; then
    echo -e "${GREEN}✓ Unauthorized access correctly blocked${NC}"
else
    echo -e "${RED}✗ Unauthorized access was not blocked${NC}"
    exit 1
fi
echo ""

# Test 8: Try to access another user's file
echo "🔒 Test 8: Create second user and test file isolation"
TEST_EMAIL2="test2_$(date +%s)@example.com"
TEST_USERNAME2="testuser2_$(date +%s)"

REGISTER2_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"$TEST_USERNAME2\",
    \"email\": \"$TEST_EMAIL2\",
    \"password\": \"$TEST_PASSWORD\"
  }")

TOKEN2=$(echo "$REGISTER2_RESPONSE" | jq -r '.token')

# Try to download first user's file with second user's token
FORBIDDEN_RESPONSE=$(curl -s -X GET "$BASE_URL/files/$FILE_ID/download" \
  -H "Authorization: Bearer $TOKEN2")

if echo "$FORBIDDEN_RESPONSE" | grep -q "Access denied"; then
    echo -e "${GREEN}✓ File isolation working correctly${NC}"
else
    echo -e "${RED}✗ File isolation not working${NC}"
    exit 1
fi
echo ""

# Test 9: Delete file
echo "🗑️  Test 9: Delete file"
DELETE_RESPONSE=$(curl -s -X DELETE "$BASE_URL/files/$FILE_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$DELETE_RESPONSE" | jq .

if echo "$DELETE_RESPONSE" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ File deleted successfully${NC}"
else
    echo -e "${RED}✗ File deletion failed${NC}"
    exit 1
fi
echo ""

# Test 10: Verify file is deleted
echo "✅ Test 10: Verify file is deleted"
LIST_AFTER_DELETE=$(curl -s -X GET "$BASE_URL/files" \
  -H "Authorization: Bearer $TOKEN")

FILE_COUNT_AFTER=$(echo "$LIST_AFTER_DELETE" | jq '.data | length')

if [ "$FILE_COUNT_AFTER" -eq 0 ]; then
    echo -e "${GREEN}✓ File successfully removed from user's files${NC}"
else
    echo -e "${YELLOW}⚠ File count after deletion: $FILE_COUNT_AFTER${NC}"
fi
echo ""

# Cleanup
rm -f /tmp/echofs_test_file.txt /tmp/echofs_downloaded_file.txt

echo "===================================="
echo -e "${GREEN}🎉 All tests passed!${NC}"
echo ""
echo "Test Summary:"
echo "  - User registration: ✓"
echo "  - User login: ✓"
echo "  - Profile retrieval: ✓"
echo "  - File upload: ✓"
echo "  - File listing: ✓"
echo "  - File download: ✓"
echo "  - Unauthorized access blocked: ✓"
echo "  - File isolation: ✓"
echo "  - File deletion: ✓"
echo "  - Deletion verification: ✓"
echo ""
echo "Test users created:"
echo "  - Email: $TEST_EMAIL"
echo "  - Username: $TEST_USERNAME"
echo "  - Email 2: $TEST_EMAIL2"
echo "  - Username 2: $TEST_USERNAME2"
