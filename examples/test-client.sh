#!/bin/bash

# Test client for the distributed job queue
# This script demonstrates how to interact with the job queue API

BASE_URL="http://localhost:8080/api/v1"

echo "🚀 Testing Distributed Job Queue"
echo "=================================="

# Test health endpoint
echo "1. Testing health endpoint..."
curl -s "$BASE_URL/../health" | jq .
echo ""

# Test job types endpoint
echo "2. Getting available job types..."
curl -s "$BASE_URL/jobs/types" | jq .
echo ""

# Test queue stats
echo "3. Getting queue statistics..."
curl -s "$BASE_URL/queue/stats" | jq .
echo ""

# Submit email job
echo "4. Submitting email job..."
EMAIL_JOB=$(curl -s -X POST "$BASE_URL/jobs" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Test Email",
      "body": "This is a test email from the job queue!"
    },
    "max_retries": 3
  }')

echo "$EMAIL_JOB" | jq .
EMAIL_JOB_ID=$(echo "$EMAIL_JOB" | jq -r .job_id)
echo "Email job ID: $EMAIL_JOB_ID"
echo ""

# Submit image processing job
echo "5. Submitting image processing job..."
IMAGE_JOB=$(curl -s -X POST "$BASE_URL/jobs" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "image_resize",
    "payload": {
      "url": "https://example.com/image.jpg",
      "width": 800,
      "height": 600,
      "format": "jpeg"
    }
  }')

echo "$IMAGE_JOB" | jq .
IMAGE_JOB_ID=$(echo "$IMAGE_JOB" | jq -r .job_id)
echo "Image job ID: $IMAGE_JOB_ID"
echo ""

# Submit math job
echo "6. Submitting math computation job..."
MATH_JOB=$(curl -s -X POST "$BASE_URL/jobs" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "math",
    "payload": {
      "operation": "fibonacci",
      "number": 30
    }
  }')

echo "$MATH_JOB" | jq .
MATH_JOB_ID=$(echo "$MATH_JOB" | jq -r .job_id)
echo "Math job ID: $MATH_JOB_ID"
echo ""

# Submit batch of jobs
echo "7. Submitting batch of jobs..."
for i in {1..5}; do
  curl -s -X POST "$BASE_URL/jobs" \
    -H "Content-Type: application/json" \
    -d "{
      \"type\": \"email\",
      \"payload\": {
        \"to\": \"batch-user-$i@example.com\",
        \"subject\": \"Batch Email #$i\",
        \"body\": \"This is batch email number $i\"
      }
    }" > /dev/null
  echo "Submitted batch job #$i"
done
echo ""

# Check queue stats after submissions
echo "8. Queue stats after job submissions..."
sleep 2
curl -s "$BASE_URL/queue/stats" | jq .
echo ""

# Test invalid job type
echo "9. Testing invalid job type (should fail)..."
INVALID_JOB=$(curl -s -X POST "$BASE_URL/jobs" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "invalid_type",
    "payload": {"test": "data"}
  }')

echo "$INVALID_JOB" | jq .
echo ""

# Test malformed request
echo "10. Testing malformed request (should fail)..."
MALFORMED_JOB=$(curl -s -X POST "$BASE_URL/jobs" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email"
  }')

echo "$MALFORMED_JOB" | jq .
echo ""

echo "🎉 Testing complete!"
echo ""
echo "💡 Tips:"
echo "- Check the worker logs to see job processing"
echo "- Monitor Redis to see queue operations: redis-cli monitor"
echo "- Use Redis Commander at http://localhost:8081 (if running with docker-compose)"