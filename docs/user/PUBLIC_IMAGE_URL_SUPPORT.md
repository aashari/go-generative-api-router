# Public Image URL Support

The Generative API Router now supports automatic downloading and conversion of public image URLs to base64 format. This feature allows you to use images from the web directly in your vision API requests without manually downloading and encoding them.

## Features

- **Automatic URL Detection**: The service automatically detects public URLs (http:// or https://)
- **Concurrent Processing**: Multiple images are processed concurrently for optimal performance
- **Format Support**: Supports PNG, JPEG, GIF, and WebP image formats
- **Size Limit**: Maximum 20MB per image
- **Mixed Content**: You can mix public URLs with base64-encoded images in the same request
- **Custom Headers**: Support for custom headers during image download (authentication, user-agent, etc.)
- **Header Removal**: Headers are automatically removed before sending to AI vendors for compatibility

## How It Works

When the service receives a vision request with image URLs:

1. It identifies all public URLs in the message content
2. Extracts any custom headers specified for each image
3. Downloads images concurrently from the URLs using the provided headers
4. Validates the content type and size
5. Converts images to base64 data URLs
6. Removes custom headers from the request structure
7. Forwards the modified request to the AI provider

## Request Format

The request format remains identical to the OpenAI vision API with optional headers support:

```json
{
  "model": "your-model",
  "messages": [{
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "What's in this image?"
      },
      {
        "type": "image_url",
        "image_url": {
          "url": "https://example.com/image.jpg",
          "headers": {
            "Authorization": "Bearer your-token",
            "User-Agent": "CustomBot/1.0"
          }
        }
      }
    ]
  }]
}
```

## Examples

### Single Public Image URL (Basic)

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-model",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Describe this image"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/320px-Cat03.jpg"
          }
        }
      ]
    }]
  }'
```

### Image with Authentication Headers

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-model",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Analyze this protected image"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://api.example.com/protected/image.jpg",
            "headers": {
              "Authorization": "Bearer your-api-token",
              "X-API-Key": "your-api-key"
            }
          }
        }
      ]
    }]
  }'
```

### Multiple Images with Different Headers

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-model",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Compare these images from different sources"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://secure-api.com/image1.jpg",
            "headers": {
              "Authorization": "Bearer token-1",
              "X-Client-ID": "client-123"
            }
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://cdn.example.com/image2.jpg",
            "headers": {
              "User-Agent": "CustomBot/2.0",
              "Referer": "https://example.com/"
            }
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://public-cdn.com/image3.jpg"
          }
        }
      ]
    }]
  }'
```

### Mixed URLs and Base64

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vision-model",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Analyze these images"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/public-image.jpg"
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://secure.example.com/private-image.jpg",
            "headers": {
              "Authorization": "Bearer secure-token"
            }
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/png;base64,iVBORw0KGgoAAAANS..."
          }
        }
      ]
    }]
  }'
```

## Custom Headers Support

### Supported Header Types

- **Authorization**: Bearer tokens, API keys, basic auth
- **User-Agent**: Custom user agent strings to avoid blocking
- **Referer**: Required by some CDNs to prevent hotlinking
- **X-API-Key**: API key headers for various services
- **Accept**: Content type preferences
- **Custom Headers**: Any custom headers required by your image sources

### Header Processing

1. **During Download**: Headers are included in the HTTP request to download the image
2. **After Processing**: Headers are automatically removed from the request structure
3. **Vendor Compatibility**: The final request sent to AI vendors contains only the standard OpenAI format
4. **Per-Image Headers**: Each image can have its own unique set of headers

### Example Header Scenarios

```json
{
  "type": "image_url",
  "image_url": {
    "url": "https://api.example.com/image.jpg",
    "headers": {
      "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "X-API-Key": "sk-1234567890abcdef",
      "User-Agent": "MyApp/1.0 (compatible; Vision API)",
      "Referer": "https://myapp.com/",
      "Accept": "image/jpeg,image/png,image/webp",
      "X-Custom-Header": "custom-value"
    }
  }
}
```

## Performance Benefits

The concurrent processing ensures that multiple images are downloaded simultaneously, significantly reducing the total processing time compared to sequential downloads.

For example, downloading 4 images:
- Sequential: ~4 seconds (1 second per image)
- Concurrent: ~1 second (all images in parallel)

Headers do not impact performance as they are processed during the same concurrent download operation.

## Error Handling

The service handles various error scenarios:

- **Invalid URL**: Returns error if the URL is malformed
- **Network Errors**: Returns error if the image cannot be downloaded
- **Authentication Errors**: Returns error if headers are invalid or insufficient
- **Invalid Content Type**: Only accepts image/* content types
- **Size Limit Exceeded**: Returns error if image exceeds 20MB
- **Timeout**: 30-second timeout per image download

### Authentication Error Example

```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "Failed to download image: status 401",
    "code": "image_download_failed"
  }
}
```

## Security Considerations

- The service only downloads from public HTTP/HTTPS URLs
- File:// and other protocols are not supported
- Downloaded images are not cached or stored
- Headers are not logged in production for security
- Each request is processed independently
- Headers are completely removed before sending to AI vendors

## Limitations

- Maximum image size: 20MB per image
- Timeout: 30 seconds per image download
- Supported formats: PNG, JPEG, GIF, WebP
- No support for authenticated URLs requiring complex OAuth flows
- Headers must be simple key-value string pairs

## Monitoring

The service logs detailed information about image processing:

```json
{
  "level": "INFO",
  "message": "Processing image URLs concurrently",
  "image_count": 3,
  "total_parts": 4
}

{
  "level": "DEBUG",
  "message": "Downloading image from URL with headers",
  "url": "https://example.com/image.jpg",
  "headers": {"Authorization": "Bearer ***", "User-Agent": "CustomBot/1.0"}
}

{
  "level": "INFO", 
  "message": "Image URL processing completed",
  "processed_count": 3,
  "error_count": 0
}
```

These logs can help monitor performance and troubleshoot issues. Note that sensitive header values are masked in production logs. 