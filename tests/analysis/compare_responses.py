#!/usr/bin/env python3
import json
import sys
import os

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
    
    matches = 0
    total = len(all_keys)
    
    for key in sorted(all_keys):
        real_info = real_structure.get(key, {"type": "MISSING", "value": None})
        router_info = router_structure.get(key, {"type": "MISSING", "value": None})
        
        match = "✓" if real_info["type"] == router_info["type"] else "✗"
        if match == "✓":
            matches += 1
        
        print(f"{key:<40} {real_info['type']:<25} {router_info['type']:<25} {match}")
    
    print(f"\nStructure Match Score: {matches}/{total} ({matches/total*100:.1f}%)")
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
    
    return matches == total

def analyze_streaming_format(filename, base_dir=""):
    """Analyze streaming response format"""
    filepath = os.path.join(base_dir, filename) if base_dir else filename
    
    print(f"\n{'='*80}")
    print(f"STREAMING FORMAT ANALYSIS: {filename}")
    print(f"{'='*80}")
    
    if not os.path.exists(filepath):
        print(f"❌ File not found: {filepath}")
        return False
    
    with open(filepath, 'r') as f:
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
    
    # Check for consistency in chunk IDs (check more chunks for better analysis)
    ids = []
    timestamps = []
    fingerprints = []
    
    for line in data_lines[:10]:  # Check first 10 chunks instead of 5
        try:
            chunk = json.loads(line[6:])
            ids.append(chunk.get('id'))
            timestamps.append(chunk.get('created'))
            fingerprints.append(chunk.get('system_fingerprint'))
        except:
            pass
    
    print(f"\nConsistency Analysis (first {len(ids)} chunks):")
    print(f"  ID consistency: {'✅' if len(set(ids)) <= 1 else '❌'} ({len(set(ids))} unique IDs)")
    print(f"  Timestamp consistency: {'✅' if len(set(timestamps)) <= 1 else '❌'} ({len(set(timestamps))} unique timestamps)")
    print(f"  Fingerprint consistency: {'✅' if len(set(fingerprints)) <= 1 else '❌'} ({len(set(fingerprints))} unique fingerprints)")
    
    if ids:
        print(f"  Sample values:")
        print(f"    ID: {ids[0]}")
        print(f"    Timestamp: {timestamps[0]}")
        print(f"    Fingerprint: {fingerprints[0]}")
    
    return True

def find_response_files():
    """Find response files in project structure"""
    # Try current directory first
    current_dir = "."
    
    # Try project root (go up two levels from tests/analysis)
    project_root = "../../"
    
    search_paths = [current_dir, project_root]
    
    file_map = {
        'openai_non_streaming': ['openai_non_streaming_response.json'],
        'router_non_streaming': ['router_non_streaming_response.json'],
        'openai_streaming': ['openai_streaming_response.txt'],
        'router_streaming': ['router_streaming_response_fixed.txt', 'router_streaming_response.txt'],
    }
    
    found_files = {}
    
    for file_type, possible_names in file_map.items():
        for search_path in search_paths:
            for filename in possible_names:
                filepath = os.path.join(search_path, filename)
                if os.path.exists(filepath):
                    found_files[file_type] = filepath
                    break
            if file_type in found_files:
                break
    
    return found_files

if __name__ == "__main__":
    print("🔍 OPENAI API COMPATIBILITY ANALYSIS")
    print("="*80)
    
    # Find available response files
    found_files = find_response_files()
    
    if not found_files:
        print("❌ No response files found. Please ensure test files exist.")
        print("\nExpected files:")
        print("  - openai_non_streaming_response.json")
        print("  - router_non_streaming_response.json") 
        print("  - openai_streaming_response.txt (optional)")
        print("  - router_streaming_response_fixed.txt (optional)")
        sys.exit(1)
    
    print(f"✅ Found {len(found_files)} test files")
    for file_type, filepath in found_files.items():
        print(f"  {file_type}: {filepath}")
    
    # Compare non-streaming responses
    if 'openai_non_streaming' in found_files and 'router_non_streaming' in found_files:
        try:
            with open(found_files['openai_non_streaming'], 'r') as f:
                real_data = json.load(f)
            
            with open(found_files['router_non_streaming'], 'r') as f:
                router_data = json.load(f)
            
            structure_match = compare_structures(real_data, router_data)
            
        except Exception as e:
            print(f"❌ Error comparing non-streaming responses: {e}")
            structure_match = False
    else:
        print("⚠️ Skipping non-streaming comparison - missing files")
        structure_match = None
    
    # Analyze streaming formats
    streaming_analyzed = False
    for file_type in ['openai_streaming', 'router_streaming']:
        if file_type in found_files:
            base_dir = os.path.dirname(found_files[file_type])
            filename = os.path.basename(found_files[file_type])
            if analyze_streaming_format(filename, base_dir):
                streaming_analyzed = True
    
    # Final assessment
    print(f"\n{'🎯 FINAL ASSESSMENT'}")
    print("="*80)
    
    if structure_match is not None:
        print(f"✅ Non-streaming structure match: {'PASS' if structure_match else 'FAIL'}")
    else:
        print(f"⚠️ Non-streaming structure match: SKIPPED")
    
    print(f"✅ Streaming format analysis: {'COMPLETED' if streaming_analyzed else 'SKIPPED'}")
    
    if structure_match:
        print(f"\n🎉 Router responses match OpenAI API structure!")
    elif structure_match is False:
        print(f"\n⚠️ Structure differences found - review output above")
    else:
        print(f"\n📋 Analysis completed with available files") 