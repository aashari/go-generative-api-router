#!/bin/bash
# Vision API example with public image URLs

API_URL="http://localhost:8082"

# Example 1: Single public image URL
echo "=== Single Public Image URL ==="
echo "The service will automatically download and convert public URLs to base64"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-analyzer",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "What is shown in this image? Describe the scene."
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/1200px-Cat03.jpg"
          }
        }
      ]
    }],
    "max_tokens": 200
  }' | jq

echo -e "\n\n"

# Example 2: Multiple public image URLs (processed concurrently)
echo "=== Multiple Public Image URLs (Concurrent Processing) ==="
echo "Multiple images are downloaded and processed concurrently for efficiency"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-compare",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Compare these two images. What are the main differences?"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/4/4d/Cat_November_2010-1a.jpg/1200px-Cat_November_2010-1a.jpg"
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/b/bb/Kittyply_edit1.jpg/1200px-Kittyply_edit1.jpg"
          }
        }
      ]
    }],
    "max_tokens": 300
  }' | jq

echo -e "\n\n"

# Example 3: Mixed URLs and base64 images
echo "=== Mixed Public URLs and Base64 Images ==="
echo "You can mix public URLs with base64-encoded images in the same request"
echo ""

# Small test image as base64 (1x1 red pixel PNG)
BASE64_IMAGE="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-mixed",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "I have uploaded a photo from the internet and a local image. Can you describe both?"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/6/68/Orange_tabby_cat_sitting_on_fallen_leaves-Hisashi-01A.jpg/1200px-Orange_tabby_cat_sitting_on_fallen_leaves-Hisashi-01A.jpg"
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

# Example 4: Error handling
echo "=== Error Handling Example ==="
echo "The service validates image URLs and handles errors gracefully"
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
          "text": "This request includes an invalid image URL"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://invalid-domain-that-does-not-exist.com/image.jpg"
          }
        }
      ]
    }]
  }' | jq

echo -e "\n\n"

# Example 5: Performance test with multiple images
echo "=== Performance Test: Multiple Images ==="
echo "Testing concurrent download of 4 images"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-performance",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Describe all four cat images briefly."
        },
        {
          "type": "image_url",
          "image_url": {"url": "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/400px-Cat03.jpg"}
        },
        {
          "type": "image_url",
          "image_url": {"url": "https://upload.wikimedia.org/wikipedia/commons/thumb/4/4d/Cat_November_2010-1a.jpg/400px-Cat_November_2010-1a.jpg"}
        },
        {
          "type": "image_url",
          "image_url": {"url": "https://upload.wikimedia.org/wikipedia/commons/thumb/b/bb/Kittyply_edit1.jpg/400px-Kittyply_edit1.jpg"}
        },
        {
          "type": "image_url",
          "image_url": {"url": "https://upload.wikimedia.org/wikipedia/commons/thumb/6/68/Orange_tabby_cat_sitting_on_fallen_leaves-Hisashi-01A.jpg/400px-Orange_tabby_cat_sitting_on_fallen_leaves-Hisashi-01A.jpg"}
        }
      ]
    }],
    "max_tokens": 400
  }' | jq

echo -e "\n"
echo "=== Notes ==="
echo "1. Public image URLs (http:// or https://) are automatically downloaded and converted to base64"
echo "2. Multiple images are processed concurrently for better performance"
echo "3. The service supports mixing public URLs with base64-encoded images"
echo "4. Image size limit is 20MB per image"
echo "5. Supported formats: PNG, JPEG, GIF, WebP"
echo "6. The original request structure is preserved - only URLs are converted" 