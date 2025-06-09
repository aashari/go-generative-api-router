# Public Image URL and File Processing Support

The Generative API Router supports automatic downloading and processing of public URLs for both images and files. This feature allows you to use images from the web directly in your vision API requests and process documents/files through text conversion without manually downloading them.

## Features

### Image Processing
- **Automatic URL Detection**: The service automatically detects public URLs (http:// or https://)
- **Concurrent Processing**: Multiple images are processed concurrently for optimal performance
- **Format Support**: Supports PNG, JPEG, GIF, and WebP image formats
- **Size Limit**: Maximum 20MB per image
- **Mixed Content**: You can mix public URLs with base64-encoded images in the same request
- **Custom Headers**: Support for custom headers during image download (authentication, user-agent, etc.)
- **Header Removal**: Headers are automatically removed before sending to AI vendors for compatibility

### File Processing
- **Document Conversion**: Automatically downloads and converts files to text using markitdown
- **Format Support**: Supports PDF, Word documents, PowerPoint, Excel, HTML, text files, and more
- **Intelligent Processing**: Uses markitdown (Microsoft's tool) for high-quality text extraction
- **Size Limit**: Maximum 20MB per file
- **Custom Headers**: Support for authentication headers during file download
- **Natural Integration**: Processed file content is seamlessly integrated into the conversation

## How It Works

### Image Processing Workflow

When the service receives a request with image URLs:

1. It identifies all public image URLs in the message content
2. Extracts any custom headers specified for each image
3. Downloads images concurrently from the URLs using the provided headers
4. Validates the content type and size
5. Converts images to base64 data URLs
6. Removes custom headers from the request structure
7. Forwards the modified request to the AI provider

### File Processing Workflow

When the service receives a request with file URLs:

1. It identifies file type content parts in the message
2. Extracts any custom headers specified for the file
3. Downloads the file from the URL using the provided headers
4. Validates the file size (max 20MB)
5. Saves the file temporarily to disk
6. Converts the file to text using markitdown
7. Replaces the file content part with a text content part containing the extracted text
8. Forwards the modified request to the AI provider

## Request Format

The request format supports both images and files with optional headers:

### Image URL Format
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

### File Processing Format
```json
{
  "model": "your-model",
  "messages": [{
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "Please analyze this document"
      },
      {
        "type": "file",
        "image_url": {
          "url": "https://example.com/document.pdf",
          "headers": {
            "Authorization": "Bearer your-token"
          }
        }
      }
    ]
  }]
}
```

### Mixed Content Format
```json
{
  "model": "your-model", 
  "messages": [{
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "Compare this image with the document"
      },
      {
        "type": "image_url",
        "image_url": {
          "url": "https://example.com/chart.png"
        }
      },
      {
        "type": "file",
        "image_url": {
          "url": "https://example.com/report.pdf"
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

### File Processing Examples

#### Basic PDF Processing

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Please summarize this research paper"
        },
        {
          "type": "file",
          "image_url": {
            "url": "https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf"
          }
        }
      ]
    }]
  }'
```

#### Protected Document with Authentication

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o", 
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Analyze this private document"
        },
        {
          "type": "file",
          "image_url": {
            "url": "https://secure-docs.example.com/report.pdf",
            "headers": {
              "Authorization": "Bearer your-access-token",
              "X-API-Key": "your-api-key"
            }
          }
        }
      ]
    }]
  }'
```

#### Mixed Content: Image + Document Analysis

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{
      "role": "user", 
      "content": [
        {
          "type": "text",
          "text": "Compare the chart with the document findings"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/sales-chart.png"
          }
        },
        {
          "type": "file",
          "image_url": {
            "url": "https://example.com/quarterly-report.pdf"
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

The concurrent processing ensures that multiple images and files are downloaded simultaneously, significantly reducing the total processing time compared to sequential downloads.

For example, downloading 4 items (images + files):
- Sequential: ~20 seconds (5 seconds per item including processing)
- Concurrent: ~5 seconds (all items processed in parallel)

Headers do not impact performance as they are processed during the same concurrent download operation. File processing time depends on the file size and complexity, with markitdown providing efficient text extraction.

## Error Handling

The service gracefully handles various error scenarios with natural AI responses:

### Image Processing Errors
- **Invalid URL**: AI responds naturally about broken/invalid image URLs
- **Network Errors**: AI explains connectivity issues and suggests alternatives
- **Authentication Errors**: AI mentions access permission issues
- **Invalid Content Type**: AI explains the URL doesn't contain a valid image
- **Size Limit Exceeded**: AI explains the image is too large (20MB limit)
- **Timeout**: AI mentions slow server response

### File Processing Errors  
- **Invalid URL**: AI responds naturally about broken/invalid file URLs
- **Network Errors**: AI explains connectivity issues for file access
- **Authentication Errors**: AI mentions file access permission issues
- **Conversion Errors**: AI explains when markitdown cannot process the file format
- **Size Limit Exceeded**: AI explains the file is too large (20MB limit)
- **Timeout**: AI mentions slow server response for file downloads

### Natural Error Responses

Instead of technical error messages, the AI responds naturally as part of the conversation. For example, if a file cannot be accessed, the AI might say: "I wasn't able to access that document. The link might be broken or require different permissions. Could you try providing the file again or check if the URL is accessible?"

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

### Image Processing
- Maximum image size: 20MB per image
- Timeout: 30 seconds per image download
- Supported formats: PNG, JPEG, GIF, WebP, BMP, TIFF, SVG
- No support for authenticated URLs requiring complex OAuth flows
- Headers must be simple key-value string pairs

### File Processing
- Maximum file size: 20MB per file
- Timeout: 30 seconds per file download
- Supported formats: PDF, Word, Excel, PowerPoint, HTML, text files, and more (via markitdown)
- Files are temporarily stored in /tmp during processing
- No support for password-protected documents
- Requires markitdown to be installed and available in PATH

## Monitoring

The service logs detailed information about image and file processing:

### Image Processing Logs
```json
{
  "level": "INFO",
  "message": "Processing image URLs and files concurrently",
  "item_count": 3,
  "total_parts": 4
}

{
  "level": "DEBUG",
  "message": "Downloading image from URL with headers",
  "url": "https://example.com/image.jpg",
  "headers": {"Authorization": "Bearer ***", "User-Agent": "CustomBot/1.0"}
}
```

### File Processing Logs
```json
{
  "level": "DEBUG",
  "message": "Downloading file from URL with headers",
  "url": "https://example.com/document.pdf",
  "headers": {"Authorization": "Bearer ***"}
}

{
  "level": "DEBUG",
  "message": "File downloaded and converted",
  "original_url": "https://example.com/document.pdf",
  "content_type": "application/pdf",
  "size_bytes": 1048576,
  "text_length": 50000
}
```

### Processing Completion
```json
{
  "level": "INFO", 
  "message": "Item processing completed with graceful error handling",
  "processed_count": 3,
  "successful_count": 2,
  "failed_count": 1,
  "total_items": 3
}
```

These logs help monitor performance and troubleshoot issues. Sensitive header values are masked in production logs, and failed items are handled gracefully with natural AI responses. 