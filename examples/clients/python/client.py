#!/usr/bin/env python3
"""Example Python client for Generative API Router"""

import requests
import json

API_BASE = "http://localhost:8082"

def chat_completion(message):
    """Send a chat completion request"""
    response = requests.post(
        f"{API_BASE}/v1/chat/completions",
        json={
            "model": "any-model",
            "messages": [{"role": "user", "content": message}]
        }
    )
    return response.json()

if __name__ == "__main__":
    result = chat_completion("Hello, how are you?")
    print(json.dumps(result, indent=2)) 