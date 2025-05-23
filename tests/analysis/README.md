# Analysis Tools

This directory contains tools for analyzing and comparing API responses between the real OpenAI API and the Generative API Router.

## Files

### `compare_responses.py`
Basic comparison tool that analyzes JSON structure and streaming format between real OpenAI API responses and router responses.

**Usage:**
```bash
python3 compare_responses.py
```

**Features:**
- Compares non-streaming JSON response structures
- Analyzes streaming response formats and consistency
- Validates field types and presence
- Checks ID consistency in streaming responses

### `comprehensive_comparison.py`
Advanced vendor compatibility verification tool that performs exhaustive testing across all vendor combinations.

**Usage:**
```bash
python3 comprehensive_comparison.py
```

**Features:**
- Cross-vendor response structure comparison (vendor=openai vs vendor=gemini)
- Streaming consistency analysis (ID, timestamp, fingerprint)
- Tool calling compatibility verification
- Model name preservation validation
- Comprehensive compatibility scoring

## Test Data Requirements

Both tools expect certain response files to be present in the root directory:
- `openai_non_streaming_response.json`
- `router_non_streaming_response.json`
- `openai_streaming_response.txt`
- `router_streaming_response_fixed.txt`
- And vendor-specific test files for comprehensive analysis

## Purpose

These tools ensure that:
1. The router maintains 100% OpenAI API compatibility
2. Both vendor=openai and vendor=gemini produce identical response structures
3. Streaming responses maintain consistent IDs, timestamps, and fingerprints
4. All advanced features (tool calling, error handling) work correctly
5. Model name transparency is preserved across all vendors 