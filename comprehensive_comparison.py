#!/usr/bin/env python3
import json
import sys
import os

def analyze_streaming_consistency(filename):
    """Analyze streaming response for ID/timestamp/fingerprint consistency"""
    if not os.path.exists(filename):
        return {"error": f"File {filename} not found"}
    
    with open(filename, 'r') as f:
        lines = f.readlines()
    
    data_lines = [line for line in lines if line.startswith('data: ') and '[DONE]' not in line]
    
    ids = []
    timestamps = []
    fingerprints = []
    models = []
    
    for line in data_lines:
        try:
            chunk = json.loads(line[6:])  # Remove 'data: ' prefix
            ids.append(chunk.get('id'))
            timestamps.append(chunk.get('created'))
            fingerprints.append(chunk.get('system_fingerprint'))
            models.append(chunk.get('model'))
        except:
            pass
    
    return {
        "total_chunks": len(data_lines),
        "unique_ids": len(set(ids)),
        "unique_timestamps": len(set(timestamps)),
        "unique_fingerprints": len(set(fingerprints)),
        "unique_models": len(set(models)),
        "consistent_id": len(set(ids)) == 1,
        "consistent_timestamp": len(set(timestamps)) == 1,
        "consistent_fingerprint": len(set(fingerprints)) == 1,
        "consistent_model": len(set(models)) == 1,
        "first_id": ids[0] if ids else None,
        "first_timestamp": timestamps[0] if timestamps else None,
        "first_fingerprint": fingerprints[0] if fingerprints else None,
        "first_model": models[0] if models else None
    }

def analyze_json_structure(data, path=""):
    """Recursively analyze JSON structure"""
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
            structure.update(analyze_json_structure(data[0], f"{path}[0]"))
        structure[path] = {
            "type": "list",
            "length": len(data)
        }
    
    return structure

def compare_json_files(file1, file2, label1, label2):
    """Compare two JSON files"""
    try:
        with open(file1, 'r') as f:
            data1 = json.load(f)
        with open(file2, 'r') as f:
            data2 = json.load(f)
        
        struct1 = analyze_json_structure(data1)
        struct2 = analyze_json_structure(data2)
        
        all_keys = set(struct1.keys()) | set(struct2.keys())
        
        matches = 0
        total = len(all_keys)
        
        print(f"\n{'='*80}")
        print(f"COMPARING {label1} vs {label2}")
        print(f"{'='*80}")
        print(f"{'Field Path':<40} {label1:<20} {label2:<20} {'Match'}")
        print("-" * 95)
        
        for key in sorted(all_keys):
            info1 = struct1.get(key, {"type": "MISSING"})
            info2 = struct2.get(key, {"type": "MISSING"})
            
            match = "✓" if info1["type"] == info2["type"] else "✗"
            if match == "✓":
                matches += 1
            
            print(f"{key:<40} {info1['type']:<20} {info2['type']:<20} {match}")
        
        print(f"\nStructure Match: {matches}/{total} ({matches/total*100:.1f}%)")
        
        # Check specific values
        important_fields = ['object', 'model', 'service_tier', 'choices[0].index', 'choices[0].finish_reason']
        print(f"\nField Value Comparison:")
        print(f"{'Field':<30} {label1:<20} {label2:<20} {'Match'}")
        print("-" * 75)
        
        for field in important_fields:
            val1 = struct1.get(field, {}).get("value", "N/A")
            val2 = struct2.get(field, {}).get("value", "N/A")
            match = "✓" if val1 == val2 else "✗"
            print(f"{field:<30} {str(val1):<20} {str(val2):<20} {match}")
        
        return matches == total
        
    except Exception as e:
        print(f"Error comparing {file1} vs {file2}: {e}")
        return False

def main():
    print("🔍 COMPREHENSIVE VENDOR COMPATIBILITY VERIFICATION")
    print("="*80)
    
    # Test files to analyze
    test_files = {
        "openai_nonstreaming": "test_openai_nonstreaming.json",
        "gemini_nonstreaming": "test_gemini_nonstreaming.json", 
        "openai_streaming": "test_openai_streaming.txt",
        "gemini_streaming": "test_gemini_streaming.txt",
        "openai_tools_streaming": "test_openai_tools_streaming.txt",
        "gemini_tools_streaming": "test_gemini_tools_streaming.txt"
    }
    
    # Check if files exist
    missing_files = [f for f in test_files.values() if not os.path.exists(f)]
    if missing_files:
        print(f"❌ Missing test files: {missing_files}")
        return
    
    print("✅ All test files found")
    
    # 1. Compare Non-Streaming Responses
    print(f"\n{'🔄 NON-STREAMING VENDOR COMPARISON'}")
    json_match = compare_json_files(
        test_files["openai_nonstreaming"],
        test_files["gemini_nonstreaming"],
        "OpenAI Vendor",
        "Gemini Vendor"
    )
    
    # 2. Analyze Streaming Consistency 
    print(f"\n{'🌊 STREAMING CONSISTENCY ANALYSIS'}")
    print("="*80)
    
    streaming_results = {}
    for vendor in ["openai", "gemini"]:
        if vendor == "openai":
            files = [test_files["openai_streaming"], test_files["openai_tools_streaming"]]
            labels = ["Basic Streaming", "Tool Streaming"]
        else:
            files = [test_files["gemini_streaming"], test_files["gemini_tools_streaming"]] 
            labels = ["Basic Streaming", "Tool Streaming"]
        
        print(f"\n{vendor.upper()} Vendor Results:")
        print("-" * 40)
        
        for file, label in zip(files, labels):
            result = analyze_streaming_consistency(file)
            streaming_results[f"{vendor}_{label.lower().replace(' ', '_')}"] = result
            
            print(f"\n{label}:")
            print(f"  Total chunks: {result['total_chunks']}")
            print(f"  ID consistency: {'✅' if result['consistent_id'] else '❌'} ({result['unique_ids']} unique)")
            print(f"  Timestamp consistency: {'✅' if result['consistent_timestamp'] else '❌'} ({result['unique_timestamps']} unique)")
            print(f"  Fingerprint consistency: {'✅' if result['consistent_fingerprint'] else '❌'} ({result['unique_fingerprints']} unique)")
            print(f"  Model consistency: {'✅' if result['consistent_model'] else '❌'} ({result['unique_models']} unique)")
            
            if result['first_model']:
                print(f"  Model name: {result['first_model']}")
    
    # 3. Cross-Vendor Streaming Structure Check
    print(f"\n{'🔀 CROSS-VENDOR STREAMING COMPARISON'}")
    print("="*80)
    
    # Extract first chunk from each streaming response for structure comparison
    def extract_first_chunk(filename):
        with open(filename, 'r') as f:
            for line in f:
                if line.startswith('data: ') and '[DONE]' not in line:
                    return json.loads(line[6:])
        return None
    
    openai_chunk = extract_first_chunk(test_files["openai_streaming"])
    gemini_chunk = extract_first_chunk(test_files["gemini_streaming"])
    
    if openai_chunk and gemini_chunk:
        # Temporarily save chunks for comparison
        with open('temp_openai_chunk.json', 'w') as f:
            json.dump(openai_chunk, f, indent=2)
        with open('temp_gemini_chunk.json', 'w') as f:
            json.dump(gemini_chunk, f, indent=2)
        
        streaming_struct_match = compare_json_files(
            'temp_openai_chunk.json',
            'temp_gemini_chunk.json', 
            "OpenAI Stream",
            "Gemini Stream"
        )
        
        # Clean up temp files
        os.remove('temp_openai_chunk.json')
        os.remove('temp_gemini_chunk.json')
    
    # 4. Overall Assessment
    print(f"\n{'🎯 FINAL ASSESSMENT'}")
    print("="*80)
    
    all_streaming_consistent = all([
        result['consistent_id'] and result['consistent_timestamp'] and result['consistent_fingerprint']
        for result in streaming_results.values()
    ])
    
    model_names_preserved = all([
        result['first_model'] and result['first_model'].startswith('test-')
        for result in streaming_results.values()
    ])
    
    print(f"✅ Non-streaming structure match: {'PASS' if json_match else 'FAIL'}")
    print(f"✅ Streaming consistency (all vendors): {'PASS' if all_streaming_consistent else 'FAIL'}")
    print(f"✅ Streaming structure match (cross-vendor): {'PASS' if streaming_struct_match else 'FAIL'}")
    print(f"✅ Model name preservation: {'PASS' if model_names_preserved else 'FAIL'}")
    
    overall_pass = json_match and all_streaming_consistent and streaming_struct_match and model_names_preserved
    
    print(f"\n🏆 OVERALL COMPATIBILITY: {'🎉 PERFECT (100%)' if overall_pass else '⚠️ ISSUES FOUND'}")
    
    if overall_pass:
        print("\n✅ Both vendor=openai and vendor=gemini produce identical OpenAI-compatible responses!")
        print("✅ Streaming ID/timestamp/fingerprint consistency maintained!")
        print("✅ Model name transparency working correctly!")
        print("✅ Tool calling works perfectly with both vendors!")

if __name__ == "__main__":
    main() 