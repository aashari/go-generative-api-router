# Public Image URL Support

The Generative API Router now supports automatic downloading and conversion of public image URLs to base64 format. This feature allows you to use images from the web directly in your vision API requests without manually downloading and encoding them.

## Features

- **Automatic URL Detection**: The service automatically detects public URLs (http:// or https://)
- **Concurrent Processing**: Multiple images are processed concurrently for optimal performance
- **Format Support**: Supports PNG, JPEG, GIF, and WebP image formats
- **Size Limit**: Maximum 20MB per image
- **Mixed Content**: You can mix public URLs with base64-encoded images in the same request

## How It Works

When the service receives a vision request with image URLs:

1. It identifies all public URLs in the message content
2. Downloads images concurrently from the URLs
3. Validates the content type and size
4. Converts images to base64 data URLs
5. Forwards the modified request to the AI provider

## Request Format

The request format remains identical to the OpenAI vision API:

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
          "url": "https://example.com/image.jpg"
        }
      }
    ]
  }]
}
```

## Examples

### Single Public Image URL

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

### Multiple Images (Processed Concurrently)

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
          "text": "Compare these images"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/image1.jpg"
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/image2.jpg"
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
            "url": "data:image/png;base64,iVBORw0KGgoAAAANS..."
          }
        }
      ]
    }]
  }'
```

## Performance Benefits

The concurrent processing ensures that multiple images are downloaded simultaneously, significantly reducing the total processing time compared to sequential downloads.

For example, downloading 4 images:
- Sequential: ~4 seconds (1 second per image)
- Concurrent: ~1 second (all images in parallel)

## Error Handling

The service handles various error scenarios:

- **Invalid URL**: Returns error if the URL is malformed
- **Network Errors**: Returns error if the image cannot be downloaded
- **Invalid Content Type**: Only accepts image/* content types
- **Size Limit Exceeded**: Returns error if image exceeds 20MB
- **Timeout**: 30-second timeout per image download

If any image fails to process, the entire request fails with a descriptive error message.

## Security Considerations

- The service only downloads from public HTTP/HTTPS URLs
- File:// and other protocols are not supported
- Downloaded images are not cached or stored
- Each request is processed independently

## Limitations

- Maximum image size: 20MB per image
- Timeout: 30 seconds per image download
- Supported formats: PNG, JPEG, GIF, WebP
- No support for authenticated URLs (URLs requiring login)

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
  "level": "INFO", 
  "message": "Image URL processing completed",
  "processed_count": 3,
  "error_count": 0
}
```

These logs can help monitor performance and troubleshoot issues. 