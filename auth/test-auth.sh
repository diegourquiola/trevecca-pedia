#!/bin/bash

# Trevecca-Pedia Auth Service - Smoke Test Script
# This script tests all the main endpoints of the auth service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
AUTH_URL="${AUTH_URL:-http://localhost:8083}"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
print_test() {
    echo -e "\n${YELLOW}TEST: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    ((TESTS_PASSED++))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    ((TESTS_FAILED++))
}

print_summary() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
    echo -e "${YELLOW}========================================${NC}"
    
    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    fi
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is not installed. Please install jq to run this script.${NC}"
    exit 1
fi

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}  Trevecca-Pedia Auth Service Tests${NC}"
echo -e "${YELLOW}  Testing: $AUTH_URL${NC}"
echo -e "${YELLOW}========================================${NC}"

# Test 1: Health Check
print_test "Health check endpoint"
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$AUTH_URL/healthz")
HEALTH_BODY=$(echo "$HEALTH_RESPONSE" | head -n -1)
HEALTH_CODE=$(echo "$HEALTH_RESPONSE" | tail -n 1)

if [ "$HEALTH_CODE" -eq 200 ]; then
    STATUS=$(echo "$HEALTH_BODY" | jq -r '.status')
    if [ "$STATUS" = "ok" ]; then
        print_success "Health check returned 200 with status: ok"
    else
        print_error "Health check returned unexpected status: $STATUS"
    fi
else
    print_error "Health check failed with code: $HEALTH_CODE"
fi

# Test 2: Login with invalid credentials
print_test "Login with invalid credentials (should fail)"
LOGIN_FAIL_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$AUTH_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"nonexistent@example.com","password":"wrongpass"}')
LOGIN_FAIL_CODE=$(echo "$LOGIN_FAIL_RESPONSE" | tail -n 1)

if [ "$LOGIN_FAIL_CODE" -eq 401 ]; then
    print_success "Invalid login correctly returned 401"
else
    print_error "Invalid login returned unexpected code: $LOGIN_FAIL_CODE"
fi

# Test 3: Login with valid credentials
print_test "Login with valid credentials"
LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$AUTH_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"dev@trevecca.edu","password":"devpass"}')
LOGIN_BODY=$(echo "$LOGIN_RESPONSE" | head -n -1)
LOGIN_CODE=$(echo "$LOGIN_RESPONSE" | tail -n 1)

if [ "$LOGIN_CODE" -eq 200 ]; then
    TOKEN=$(echo "$LOGIN_BODY" | jq -r '.accessToken')
    USER_EMAIL=$(echo "$LOGIN_BODY" | jq -r '.user.email')
    USER_ROLES=$(echo "$LOGIN_BODY" | jq -r '.user.roles | join(",")')
    
    if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
        print_success "Login successful, received token"
        echo "  Email: $USER_EMAIL"
        echo "  Roles: $USER_ROLES"
    else
        print_error "Login returned 200 but no valid token"
    fi
else
    print_error "Login failed with code: $LOGIN_CODE"
    echo "$LOGIN_BODY"
fi

# Test 4: Access /auth/me without token
print_test "Access /auth/me without token (should fail)"
ME_NOAUTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$AUTH_URL/auth/me")
ME_NOAUTH_CODE=$(echo "$ME_NOAUTH_RESPONSE" | tail -n 1)

if [ "$ME_NOAUTH_CODE" -eq 401 ]; then
    print_success "Unauthenticated request correctly returned 401"
else
    print_error "Unauthenticated request returned unexpected code: $ME_NOAUTH_CODE"
fi

# Test 5: Access /auth/me with invalid token
print_test "Access /auth/me with invalid token (should fail)"
ME_BADTOKEN_RESPONSE=$(curl -s -w "\n%{http_code}" "$AUTH_URL/auth/me" \
    -H "Authorization: Bearer invalid.token.here")
ME_BADTOKEN_CODE=$(echo "$ME_BADTOKEN_RESPONSE" | tail -n 1)

if [ "$ME_BADTOKEN_CODE" -eq 401 ]; then
    print_success "Invalid token correctly returned 401"
else
    print_error "Invalid token returned unexpected code: $ME_BADTOKEN_CODE"
fi

# Test 6: Access /auth/me with valid token
if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
    print_test "Access /auth/me with valid token"
    ME_RESPONSE=$(curl -s -w "\n%{http_code}" "$AUTH_URL/auth/me" \
        -H "Authorization: Bearer $TOKEN")
    ME_BODY=$(echo "$ME_RESPONSE" | head -n -1)
    ME_CODE=$(echo "$ME_RESPONSE" | tail -n 1)
    
    if [ "$ME_CODE" -eq 200 ]; then
        ME_EMAIL=$(echo "$ME_BODY" | jq -r '.email')
        ME_ROLES=$(echo "$ME_BODY" | jq -r '.roles | join(",")')
        
        if [ "$ME_EMAIL" = "dev@trevecca.edu" ]; then
            print_success "Successfully retrieved user info from token"
            echo "  Email: $ME_EMAIL"
            echo "  Roles: $ME_ROLES"
        else
            print_error "Retrieved user info but email doesn't match"
        fi
    else
        print_error "/auth/me failed with code: $ME_CODE"
        echo "$ME_BODY"
    fi
else
    print_error "Skipping /auth/me test - no valid token from login"
fi

# Test 7: Login with missing fields
print_test "Login with missing password (should fail)"
LOGIN_MISSING_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$AUTH_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"dev@trevecca.edu"}')
LOGIN_MISSING_CODE=$(echo "$LOGIN_MISSING_RESPONSE" | tail -n 1)

if [ "$LOGIN_MISSING_CODE" -eq 400 ]; then
    print_success "Missing password correctly returned 400"
else
    print_error "Missing password returned unexpected code: $LOGIN_MISSING_CODE"
fi

# Test 8: Login with invalid email format
print_test "Login with invalid email format (should fail)"
LOGIN_BADEMAIL_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$AUTH_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"not-an-email","password":"test"}')
LOGIN_BADEMAIL_CODE=$(echo "$LOGIN_BADEMAIL_RESPONSE" | tail -n 1)

if [ "$LOGIN_BADEMAIL_CODE" -eq 400 ]; then
    print_success "Invalid email format correctly returned 400"
else
    print_error "Invalid email format returned unexpected code: $LOGIN_BADEMAIL_CODE"
fi

# Test 9: Token contains expected claims
if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
    print_test "Token structure validation"
    
    # Decode JWT (basic check - just count segments)
    TOKEN_SEGMENTS=$(echo "$TOKEN" | tr '.' '\n' | wc -l)
    
    if [ "$TOKEN_SEGMENTS" -eq 3 ]; then
        print_success "Token has correct structure (3 segments)"
    else
        print_error "Token has incorrect structure: $TOKEN_SEGMENTS segments"
    fi
fi

# Print summary
print_summary
