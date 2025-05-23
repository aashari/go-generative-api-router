#!/usr/bin/env python3
"""
Utility script to generate test streaming response data for validation.
This helps create sample data to test the streaming consistency analysis.
"""

import json
import sys
import argparse

def generate_streaming_response(conversation_id, model_name, chunks_count=5, consistent=True):
    """Generate a streaming response with consistent or inconsistent metadata"""
    
    if consistent:
        # Generate consistent values
        timestamp = 1703123456
        fingerprint = "fp_abcd123456"
        response_id = conversation_id
    
    chunks = []
    
    for i in range(chunks_count):
        if not consistent:
            # Generate different values for each chunk (simulating the bug)
            timestamp = 1703123456 + i
            fingerprint = f"fp_abcd12345{i}"
            response_id = f"chatcmpl-test{i:03d}"
        
        # First chunk with role
        if i == 0:
            chunk = {
                "id": response_id,
                "object": "chat.completion.chunk",
                "created": timestamp,
                "model": model_name,
                "system_fingerprint": fingerprint,
                "choices": [{
                    "index": 0,
                    "delta": {
                        "role": "assistant",
                        "content": ""
                    },
                    "logprobs": None,
                    "finish_reason": None
                }]
            }
        # Middle chunks with content
        elif i < chunks_count - 1:
            chunk = {
                "id": response_id,
                "object": "chat.completion.chunk", 
                "created": timestamp,
                "model": model_name,
                "system_fingerprint": fingerprint,
                "choices": [{
                    "index": 0,
                    "delta": {
                        "content": f"word{i} "
                    },
                    "logprobs": None,
                    "finish_reason": None
                }]
            }
        # Final chunk
        else:
            chunk = {
                "id": response_id,
                "object": "chat.completion.chunk",
                "created": timestamp,
                "model": model_name,
                "system_fingerprint": fingerprint,
                "choices": [{
                    "index": 0,
                    "delta": {},
                    "logprobs": None,
                    "finish_reason": "stop"
                }]
            }
        
        chunks.append(f"data: {json.dumps(chunk)}\n\n")
    
    # Add the final [DONE] marker
    chunks.append("data: [DONE]\n\n")
    
    return chunks

def generate_tool_calling_stream(conversation_id, model_name, consistent=True):
    """Generate a streaming response with tool calling"""
    
    if consistent:
        timestamp = 1703123456
        fingerprint = "fp_abcd123456"
        response_id = conversation_id
    
    chunks = []
    
    # First chunk with role
    if not consistent:
        timestamp = 1703123456
        fingerprint = "fp_abcd123456"
        response_id = f"chatcmpl-tool001"
    
    chunk1 = {
        "id": response_id,
        "object": "chat.completion.chunk",
        "created": timestamp,
        "model": model_name,
        "system_fingerprint": fingerprint,
        "choices": [{
            "index": 0,
            "delta": {
                "role": "assistant",
                "content": None,
                "tool_calls": [{
                    "index": 0,
                    "id": "call_test123",
                    "type": "function",
                    "function": {
                        "name": "get_weather",
                        "arguments": ""
                    }
                }]
            },
            "logprobs": None,
            "finish_reason": None
        }]
    }
    chunks.append(f"data: {json.dumps(chunk1)}\n\n")
    
    # Tool call arguments chunk
    if not consistent:
        timestamp = 1703123457
        fingerprint = "fp_abcd123457"
        response_id = f"chatcmpl-tool002"
    
    chunk2 = {
        "id": response_id,
        "object": "chat.completion.chunk",
        "created": timestamp,
        "model": model_name,
        "system_fingerprint": fingerprint,
        "choices": [{
            "index": 0,
            "delta": {
                "tool_calls": [{
                    "index": 0,
                    "function": {
                        "arguments": '{"location": "Boston"}'
                    }
                }]
            },
            "logprobs": None,
            "finish_reason": None
        }]
    }
    chunks.append(f"data: {json.dumps(chunk2)}\n\n")
    
    # Final chunk
    if not consistent:
        timestamp = 1703123458
        fingerprint = "fp_abcd123458"
        response_id = f"chatcmpl-tool003"
    
    chunk3 = {
        "id": response_id,
        "object": "chat.completion.chunk",
        "created": timestamp,
        "model": model_name,
        "system_fingerprint": fingerprint,
        "choices": [{
            "index": 0,
            "delta": {},
            "logprobs": None,
            "finish_reason": "tool_calls"
        }]
    }
    chunks.append(f"data: {json.dumps(chunk3)}\n\n")
    
    chunks.append("data: [DONE]\n\n")
    
    return chunks

def main():
    parser = argparse.ArgumentParser(description='Generate test streaming response data')
    parser.add_argument('--output', '-o', default='test_streaming.txt',
                       help='Output filename (default: test_streaming.txt)')
    parser.add_argument('--model', default='test-model-v1',
                       help='Model name to use (default: test-model-v1)')
    parser.add_argument('--inconsistent', action='store_true',
                       help='Generate inconsistent metadata (simulate bug)')
    parser.add_argument('--tool-calling', action='store_true',
                       help='Generate tool calling example')
    parser.add_argument('--chunks', type=int, default=5,
                       help='Number of content chunks (default: 5)')
    
    args = parser.parse_args()
    
    conversation_id = "chatcmpl-test123456"
    consistent = not args.inconsistent
    
    print(f"Generating {'consistent' if consistent else 'inconsistent'} streaming data...")
    print(f"Model: {args.model}")
    print(f"Output: {args.output}")
    
    if args.tool_calling:
        print("Type: Tool calling")
        chunks = generate_tool_calling_stream(conversation_id, args.model, consistent)
    else:
        print(f"Type: Basic streaming ({args.chunks} content chunks)")
        chunks = generate_streaming_response(conversation_id, args.model, args.chunks, consistent)
    
    with open(args.output, 'w') as f:
        f.writelines(chunks)
    
    print(f"✅ Generated {len(chunks)} chunks")
    print(f"💾 Saved to: {args.output}")
    
    # Quick validation
    print("\n📊 Quick validation:")
    ids = []
    timestamps = []
    fingerprints = []
    
    for chunk_line in chunks:
        if chunk_line.startswith('data: ') and '[DONE]' not in chunk_line:
            try:
                chunk_data = json.loads(chunk_line[6:])
                ids.append(chunk_data.get('id'))
                timestamps.append(chunk_data.get('created'))
                fingerprints.append(chunk_data.get('system_fingerprint'))
            except:
                pass
    
    print(f"  ID consistency: {'✅' if len(set(ids)) == 1 else '❌'} ({len(set(ids))} unique)")
    print(f"  Timestamp consistency: {'✅' if len(set(timestamps)) == 1 else '❌'} ({len(set(timestamps))} unique)")
    print(f"  Fingerprint consistency: {'✅' if len(set(fingerprints)) == 1 else '❌'} ({len(set(fingerprints))} unique)")

if __name__ == "__main__":
    main() 