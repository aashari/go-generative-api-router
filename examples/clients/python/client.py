#!/usr/bin/env python3
"""Example Python client for Generative API Router"""

import requests
import json

API_BASE = "http://localhost:8082"

def chat_completion(message):
    """Send a basic chat completion request"""
    response = requests.post(
        f"{API_BASE}/v1/chat/completions",
        json={
            "model": "any-model",
            "messages": [{"role": "user", "content": message}]
        }
    )
    return response.json()

def process_file(file_url, question, headers=None):
    """Process a file and ask a question about it"""
    content = [
        {
            "type": "text",
            "text": question
        },
        {
            "type": "file_url",
            "file_url": {
                "url": file_url
            }
        }
    ]
    
    # Add custom headers if provided
    if headers:
        content[1]["file_url"]["headers"] = headers
    
    response = requests.post(
        f"{API_BASE}/v1/chat/completions",
        json={
            "model": "document-analyzer",
            "messages": [{"role": "user", "content": content}]
        }
    )
    return response.json()

def process_multiple_files(file_urls, question):
    """Process multiple files and ask a question about them"""
    content = [{"type": "text", "text": question}]
    
    # Add each file to the content
    for url in file_urls:
        content.append({
            "type": "file_url",
            "file_url": {"url": url}
        })
    
    response = requests.post(
        f"{API_BASE}/v1/chat/completions",
        json={
            "model": "multi-file-analyzer",
            "messages": [{"role": "user", "content": content}]
        }
    )
    return response.json()

if __name__ == "__main__":
    print("=== Basic Chat Example ===")
    result = chat_completion("Hello, how are you?")
    print(json.dumps(result, indent=2))
    
    print("\n=== File Processing Example ===")
    # Example with Apple's research paper
    file_result = process_file(
        "https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf",
        "Please provide a brief summary of this research paper."
    )
    print(json.dumps(file_result, indent=2))
    
    print("\n=== Multiple Files Example ===")
    # Example with multiple files (note: using same file twice for demo)
    multi_result = process_multiple_files([
        "https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf",
        "https://example.com/another-document.pdf"  # Would need a real URL
    ], "Compare these two documents.")
    print(json.dumps(multi_result, indent=2)) 