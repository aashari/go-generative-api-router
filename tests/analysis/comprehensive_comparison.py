#!/usr/bin/env python3
import json
import sys
import os
import argparse

def analyze_streaming_consistency(filename, base_dir=""):
    """Analyze streaming response for ID/timestamp/fingerprint consistency"""
    filepath = os.path.join(base_dir, filename) if base_dir else filename
    
    if not os.path.exists(filepath):
        return {"error": f"File {filepath} not found"}
    
    with open(filepath, 'r') as f:
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
        if not os.path.exists(file1):
            print(f"❌ File not found: {file1}")
            return False
        if not os.path.exists(file2):
            print(f"❌ File not found: {file2}")
            return False
            
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
        print(f"❌ Error comparing {file1} vs {file2}: {e}")
        return False

def find_test_files(data_dir="../../"):
    """Find available test files in the specified directory"""
    test_file_patterns = {
        "openai_nonstreaming": ["test_openai_nonstreaming.json", "openai_non_streaming_response.json"],
        "gemini_nonstreaming": ["test_gemini_nonstreaming.json", "gemini_non_streaming_response.json"], 
        "openai_streaming": ["test_openai_streaming.txt", "openai_streaming_response.txt"],
        "gemini_streaming": ["test_gemini_streaming.txt", "gemini_streaming_response.txt"],
        "openai_tools_streaming": ["test_openai_tools_streaming.txt"],
        "gemini_tools_streaming": ["test_gemini_tools_streaming.txt"],
        "router_nonstreaming": ["router_non_streaming_response.json", "router_non_streaming_test.json"],
        "router_streaming": ["router_streaming_response_fixed.txt", "router_streaming_response.txt"]
    }
    
    found_files = {}
    
    for file_type, possible_names in test_file_patterns.items():
        for filename in possible_names:
            filepath = os.path.join(data_dir, filename)
            if os.path.exists(filepath):
                found_files[file_type] = filepath
                break
    
    return found_files

def main():
    parser = argparse.ArgumentParser(description='Comprehensive vendor compatibility verification')
    parser.add_argument('--data-dir', default='../../', 
                       help='Directory containing test files (default: ../../)')
    parser.add_argument('--verbose', action='store_true',
                       help='Enable verbose output')
    
    args = parser.parse_args()
    
    print("🔍 COMPREHENSIVE VENDOR COMPATIBILITY VERIFICATION")
    print("="*80)
    
    # Find available test files
    found_files = find_test_files(args.data_dir)
    
    if not found_files:
        print("❌ No test files found in the specified directory")
        print(f"Searched in: {os.path.abspath(args.data_dir)}")
        print("\nExpected files (any of these patterns):")
        print("  - test_openai_nonstreaming.json / openai_non_streaming_response.json")
        print("  - test_gemini_nonstreaming.json")
        print("  - test_openai_streaming.txt / openai_streaming_response.txt")
        print("  - test_gemini_streaming.txt")
        print("  - router_non_streaming_response.json")
        return 1
    
    print(f"✅ Found {len(found_files)} test files:")
    for file_type, filepath in found_files.items():
        rel_path = os.path.relpath(filepath)
        print(f"  {file_type}: {rel_path}")
    
    # Track results for final assessment
    results = {
        'json_match': None,
        'streaming_consistent': None,
        'streaming_struct_match': None,
        'model_names_preserved': None
    }
    
    # 1. Compare Non-Streaming Responses (if available)
    if 'openai_nonstreaming' in found_files and 'gemini_nonstreaming' in found_files:
        print(f"\n{'🔄 NON-STREAMING VENDOR COMPARISON'}")
        results['json_match'] = compare_json_files(
            found_files["openai_nonstreaming"],
            found_files["gemini_nonstreaming"],
            "OpenAI Vendor",
            "Gemini Vendor"
        )
    elif 'openai_nonstreaming' in found_files and 'router_nonstreaming' in found_files:
        print(f"\n{'🔄 NON-STREAMING COMPARISON (OpenAI vs Router)'}")
        results['json_match'] = compare_json_files(
            found_files["openai_nonstreaming"],
            found_files["router_nonstreaming"],
            "Real OpenAI",
            "Router"
        )
    else:
        print(f"\n{'⚠️ SKIPPING NON-STREAMING COMPARISON - missing files'}")
    
    # 2. Analyze Streaming Consistency 
    print(f"\n{'🌊 STREAMING CONSISTENCY ANALYSIS'}")
    print("="*80)
    
    streaming_results = {}
    all_streaming_consistent = True
    model_names_preserved = True
    
    # Check available streaming files
    streaming_files = {
        'openai_basic': found_files.get('openai_streaming'),
        'gemini_basic': found_files.get('gemini_streaming'), 
        'openai_tools': found_files.get('openai_tools_streaming'),
        'gemini_tools': found_files.get('gemini_tools_streaming'),
        'router_basic': found_files.get('router_streaming')
    }
    
    for file_key, filepath in streaming_files.items():
        if filepath:
            base_dir = os.path.dirname(filepath)
            filename = os.path.basename(filepath)
            result = analyze_streaming_consistency(filename, base_dir)
            streaming_results[file_key] = result
            
            print(f"\n{file_key.replace('_', ' ').title()}:")
            print(f"  File: {os.path.relpath(filepath)}")
            
            if 'error' in result:
                print(f"  ❌ {result['error']}")
                continue
                
            print(f"  Total chunks: {result['total_chunks']}")
            print(f"  ID consistency: {'✅' if result['consistent_id'] else '❌'} ({result['unique_ids']} unique)")
            print(f"  Timestamp consistency: {'✅' if result['consistent_timestamp'] else '❌'} ({result['unique_timestamps']} unique)")
            print(f"  Fingerprint consistency: {'✅' if result['consistent_fingerprint'] else '❌'} ({result['unique_fingerprints']} unique)")
            print(f"  Model consistency: {'✅' if result['consistent_model'] else '❌'} ({result['unique_models']} unique)")
            
            if result['first_model']:
                print(f"  Model name: {result['first_model']}")
                # Check if model name looks like a test model
                if not (result['first_model'].startswith('test-') or 
                       result['first_model'].startswith('gpt-') or
                       result['first_model'].startswith('gemini-')):
                    model_names_preserved = False
            
            # Check if streaming is consistent
            if not (result['consistent_id'] and result['consistent_timestamp'] and result['consistent_fingerprint']):
                all_streaming_consistent = False
    
    results['streaming_consistent'] = all_streaming_consistent
    results['model_names_preserved'] = model_names_preserved
    
    # 3. Cross-Vendor Streaming Structure Check (if we have both vendors)
    openai_streaming = found_files.get('openai_streaming') or found_files.get('router_streaming')
    gemini_streaming = found_files.get('gemini_streaming')
    
    if openai_streaming and gemini_streaming:
        print(f"\n{'🔀 CROSS-VENDOR STREAMING COMPARISON'}")
        print("="*80)
        
        # Extract first chunk from each streaming response for structure comparison
        def extract_first_chunk(filepath):
            with open(filepath, 'r') as f:
                for line in f:
                    if line.startswith('data: ') and '[DONE]' not in line:
                        return json.loads(line[6:])
            return None
        
        openai_chunk = extract_first_chunk(openai_streaming)
        gemini_chunk = extract_first_chunk(gemini_streaming)
        
        if openai_chunk and gemini_chunk:
            # Temporarily save chunks for comparison
            temp_dir = "/tmp" if os.path.exists("/tmp") else "."
            temp_openai = os.path.join(temp_dir, 'temp_openai_chunk.json')
            temp_gemini = os.path.join(temp_dir, 'temp_gemini_chunk.json')
            
            with open(temp_openai, 'w') as f:
                json.dump(openai_chunk, f, indent=2)
            with open(temp_gemini, 'w') as f:
                json.dump(gemini_chunk, f, indent=2)
            
            results['streaming_struct_match'] = compare_json_files(
                temp_openai,
                temp_gemini, 
                "OpenAI Stream",
                "Gemini Stream"
            )
            
            # Clean up temp files
            try:
                os.remove(temp_openai)
                os.remove(temp_gemini)
            except:
                pass
        else:
            print("❌ Could not extract streaming chunks for comparison")
            results['streaming_struct_match'] = False
    else:
        print(f"\n{'⚠️ SKIPPING CROSS-VENDOR STREAMING COMPARISON - missing files'}")
    
    # 4. Overall Assessment
    print(f"\n{'🎯 FINAL ASSESSMENT'}")
    print("="*80)
    
    if results['json_match'] is not None:
        print(f"✅ Non-streaming structure match: {'PASS' if results['json_match'] else 'FAIL'}")
    else:
        print(f"⚠️ Non-streaming structure match: SKIPPED")
        
    print(f"✅ Streaming consistency (all files): {'PASS' if results['streaming_consistent'] else 'FAIL'}")
    
    if results['streaming_struct_match'] is not None:
        print(f"✅ Streaming structure match (cross-vendor): {'PASS' if results['streaming_struct_match'] else 'FAIL'}")
    else:
        print(f"⚠️ Streaming structure match: SKIPPED")
        
    print(f"✅ Model name preservation: {'PASS' if results['model_names_preserved'] else 'FAIL'}")
    
    # Calculate overall score
    passed_tests = sum(1 for r in results.values() if r is True)
    total_tests = sum(1 for r in results.values() if r is not None)
    skipped_tests = sum(1 for r in results.values() if r is None)
    
    if total_tests > 0:
        overall_score = (passed_tests / total_tests) * 100
        print(f"\n🏆 OVERALL COMPATIBILITY: {overall_score:.1f}% ({passed_tests}/{total_tests} tests passed)")
        
        if skipped_tests > 0:
            print(f"📋 {skipped_tests} tests skipped due to missing files")
        
        if overall_score == 100:
            print("\n✅ Perfect OpenAI API compatibility achieved!")
            print("✅ Both vendor responses produce identical structures!")
            print("✅ Streaming consistency maintained across all vendors!")
            return 0
        elif overall_score >= 80:
            print(f"\n⚠️ Good compatibility with minor issues")
            return 1
        else:
            print(f"\n❌ Significant compatibility issues found")
            return 2
    else:
        print(f"\n📋 Analysis completed but no tests could be performed")
        print("Please ensure test files are available for proper validation")
        return 3

if __name__ == "__main__":
    sys.exit(main()) 