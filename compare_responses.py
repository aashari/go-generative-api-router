#!/usr/bin/env python3
import json
import sys

def analyze_json_structure(data, path=""):
    """Recursively analyze the structure of JSON data"""
    structure = {}
    
    if isinstance(data, dict):
        for key, value in data.items():
            current_path = f"{path}.{key}" if path else key
            structure[current_path] = {
                "type": type(value).__name__,
                "value": value if not isinstance(value, (dict, list)) else None
            }
            if isinstance(value, (dict, list)):
                structure.update(analyze_json_structure(value, current_path))
    elif isinstance(data, list):
        if data:
            # Analyze first item in list as representative
            structure.update(analyze_json_structure(data[0], f"{path}[0]"))
        structure[path] = {
            "type": "list",
            "length": len(data)
        }
    
    return structure

def compare_structures(real_data, router_data):
    """Compare the structures of real and router API responses"""
    print("=" * 80)
    print("RESPONSE STRUCTURE COMPARISON")
    print("=" * 80)
    
    real_structure = analyze_json_structure(real_data)
    router_structure = analyze_json_structure(router_data)
    
    all_keys = set(real_structure.keys()) | set(router_structure.keys())
    
    print(f"{'Field Path':<40} {'Real API':<25} {'Router API':<25} {'Match'}")
    print("-" * 95)
    
    for key in sorted(all_keys):
        real_info = real_structure.get(key, {"type": "MISSING", "value": None})
        router_info = router_structure.get(key, {"type": "MISSING", "value": None})
        
        match = "✓" if real_info["type"] == router_info["type"] else "✗"
        
        print(f"{key:<40} {real_info['type']:<25} {router_info['type']:<25} {match}")
    
    print("\n" + "=" * 80)
    print("FIELD VALUE COMPARISON (for non-content fields)")
    print("=" * 80)
    
    # Compare specific important fields
    important_fields = ['object', 'model', 'service_tier', 'choices[0].index', 'choices[0].finish_reason']
    
    for field in important_fields:
        real_val = real_structure.get(field, {}).get("value", "MISSING")
        router_val = router_structure.get(field, {}).get("value", "MISSING")
        match = "✓" if real_val == router_val else "✗"
        print(f"{field:<40} {str(real_val):<25} {str(router_val):<25} {match}")

def analyze_streaming_format(filename):
    """Analyze streaming response format"""
    print(f"\n{'='*80}")
    print(f"STREAMING FORMAT ANALYSIS: {filename}")
    print(f"{'='*80}")
    
    with open(filename, 'r') as f:
        lines = f.readlines()
    
    data_lines = [line for line in lines if line.startswith('data: ')]
    
    print(f"Total lines: {len(lines)}")
    print(f"Data lines: {len(data_lines)}")
    print(f"Empty lines: {len([l for l in lines if l.strip() == ''])}")
    print(f"Other lines: {len(lines) - len(data_lines) - len([l for l in lines if l.strip() == ''])}")
    
    if data_lines:
        print("\nFirst data chunk structure:")
        try:
            first_chunk = json.loads(data_lines[0][6:])  # Remove 'data: ' prefix
            for key, value in first_chunk.items():
                if key != 'choices':  # Skip content analysis
                    print(f"  {key}: {type(value).__name__}")
                else:
                    print(f"  {key}: list[{len(value)}]")
                    if value:
                        print(f"    choices[0] keys: {list(value[0].keys())}")
        except Exception as e:
            print(f"  Error parsing first chunk: {e}")
    
    # Check for consistency in chunk IDs
    ids = []
    for line in data_lines[:5]:  # Check first 5 chunks
        try:
            chunk = json.loads(line[6:])
            ids.append(chunk.get('id'))
        except:
            pass
    
    print(f"\nID consistency in first 5 chunks:")
    print(f"  Unique IDs: {len(set(ids))}")
    print(f"  IDs: {ids}")

if __name__ == "__main__":
    # Compare non-streaming responses
    try:
        with open('openai_non_streaming_response.json', 'r') as f:
            real_data = json.load(f)
        
        with open('router_non_streaming_response.json', 'r') as f:
            router_data = json.load(f)
        
        compare_structures(real_data, router_data)
        
    except FileNotFoundError as e:
        print(f"Error: {e}")
        print("Make sure both response files exist")
    
    # Analyze streaming formats
    try:
        analyze_streaming_format('openai_streaming_response.txt')
        analyze_streaming_format('router_streaming_response_fixed.txt')
    except FileNotFoundError as e:
        print(f"Error analyzing streaming: {e}") 