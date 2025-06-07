#!/bin/bash
# Vision API example with custom headers for image downloads

API_URL="http://localhost:8082"

echo "=== Vision API with Custom Headers for Image Downloads ==="
echo ""
echo "This example demonstrates how to provide custom headers when downloading images from URLs."
echo "Headers are used during the download process but are automatically removed before sending to vendors."
echo ""

# Example 1: Image with authentication header
echo "=== Example 1: Image with Authentication Header ==="
echo "Downloading an image that requires authentication"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-auth-test",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "What do you see in this authenticated image?"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/protected/image.jpg",
            "headers": {
              "Authorization": "Bearer your-api-token-here",
              "X-API-Key": "your-api-key-here"
            }
          }
        }
      ]
    }],
    "max_tokens": 200
  }' | jq

echo -e "\n\n"

# Example 2: Image with custom user agent and referer
echo "=== Example 2: Image with Custom User Agent and Referer ==="
echo "Some image servers require specific headers to prevent hotlinking"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-custom-headers",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Describe this image that requires custom headers"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/hotlink-protected/image.png",
            "headers": {
              "User-Agent": "Mozilla/5.0 (compatible; MyBot/1.0)",
              "Referer": "https://example.com/",
              "Accept": "image/*"
            }
          }
        }
      ]
    }],
    "max_tokens": 150
  }' | jq

echo -e "\n\n"

# Example 3: Multiple images with different headers
echo "=== Example 3: Multiple Images with Different Headers ==="
echo "Each image can have its own set of custom headers"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-multi-headers",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Compare these images from different sources with different authentication requirements"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://api.example.com/images/1.jpg",
            "headers": {
              "Authorization": "Bearer token-for-api-example",
              "X-Client-ID": "client-123"
            }
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://cdn.another-site.com/protected/2.jpg",
            "headers": {
              "X-API-Key": "different-api-key",
              "User-Agent": "CustomBot/2.0"
            }
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://public-cdn.com/open/3.jpg"
          }
        }
      ]
    }],
    "max_tokens": 300
  }' | jq

echo -e "\n\n"

# Example 4: Mixed content with headers and base64
echo "=== Example 4: Mixed Content - Headers, URLs, and Base64 ==="
echo "Combining images with headers, public URLs, and base64 data"
echo ""

# Small test image as base64 (1x1 red pixel PNG)
BASE64_IMAGE="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-mixed-sources",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Analyze these images from different sources: authenticated URL, public URL, and base64 data"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://secure.example.com/private/image.jpg",
            "headers": {
              "Authorization": "Bearer secure-token",
              "X-Source": "secure-api"
            }
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/400px-Cat03.jpg"
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/png;base64,'$BASE64_IMAGE'"
          }
        }
      ]
    }],
    "max_tokens": 250
  }' | jq

echo -e "\n\n"

# Example 5: Error handling with invalid headers
echo "=== Example 5: Error Handling - Invalid Authentication ==="
echo "Demonstrating error handling when headers are incorrect"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-error-test",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "This should fail due to invalid authentication"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://httpbin.org/status/401",
            "headers": {
              "Authorization": "Bearer invalid-token"
            }
          }
        }
      ]
    }]
  }' | jq

echo -e "\n"
echo "=== Key Features ==="
echo "1. Custom headers are used during image download (authentication, user-agent, etc.)"
echo "2. Headers are automatically removed before sending request to AI vendors"
echo "3. Each image can have its own set of headers"
echo "4. Works with concurrent image processing"
echo "5. Compatible with existing base64 and public URL functionality"
echo "6. Comprehensive error handling for authentication failures"
echo ""
echo "=== Supported Header Types ==="
echo "- Authorization: Bearer tokens, API keys"
echo "- User-Agent: Custom user agent strings"
echo "- Referer: Required by some CDNs"
echo "- X-API-Key: API key headers"
echo "- Accept: Content type preferences"
echo "- Any custom headers required by your image sources" 