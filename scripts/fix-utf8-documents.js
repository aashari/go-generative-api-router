// MongoDB script to fix documents with invalid UTF-8 characters
// Run this script in MongoDB shell or MongoDB Compass

// List of problematic document IDs identified
const problematicIds = [
  "684c6726f397fa5f1099fa54",
  "684c6832f397fa5f1099fa55",
  "684c695edf578e40ebfbe11a",
  "684c6ad8f397fa5f1099fa5d",
  "684c6cc4f397fa5f1099fa64",
];

// Function to sanitize UTF-8 strings
function sanitizeUTF8String(str) {
  if (typeof str !== "string") return str;

  // Replace invalid UTF-8 characters with replacement character
  return str.replace(
    /[\uFFFD\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F]/g,
    "�"
  );
}

// Function to recursively sanitize an object
function sanitizeObject(obj) {
  if (obj === null || obj === undefined) return obj;

  if (typeof obj === "string") {
    return sanitizeUTF8String(obj);
  }

  if (Array.isArray(obj)) {
    return obj.map((item) => sanitizeObject(item));
  }

  if (typeof obj === "object") {
    const sanitized = {};
    for (const [key, value] of Object.entries(obj)) {
      const sanitizedKey = sanitizeUTF8String(key);
      sanitized[sanitizedKey] = sanitizeObject(value);
    }
    return sanitized;
  }

  return obj;
}

// Process each problematic document
print("Starting UTF-8 sanitization for problematic documents...");

problematicIds.forEach(function (id) {
  try {
    print("Processing document ID: " + id);

    // Try to read the document with projection to avoid UTF-8 errors
    const doc = db["generative-usages"].findOne(
      { _id: ObjectId(id) },
      { _id: 1, vendor: 1, status_code: 1, created_at: 1, request_id: 1 }
    );

    if (!doc) {
      print("Document not found: " + id);
      return;
    }

    print(
      "Found document - Vendor: " +
        doc.vendor +
        ", Status: " +
        doc.status_code +
        ", RequestID: " +
        doc.request_id
    );

    // Since we can't read the full document due to UTF-8 issues,
    // we'll delete and recreate it with sanitized data
    // First, let's mark it for manual review by adding a flag

    try {
      db["generative-usages"].updateOne(
        { _id: ObjectId(id) },
        { $set: { utf8_issue_detected: true, needs_manual_review: true } }
      );
      print("Marked document for manual review: " + id);
    } catch (updateError) {
      print("Cannot update document due to UTF-8 issues: " + id);
      print("Recommendation: Delete this document manually");
    }
  } catch (error) {
    print("Error processing document " + id + ": " + error.message);
    print("Recommendation: Delete this document manually");
  }
});

print("UTF-8 sanitization process completed.");
print("");
print("NEXT STEPS:");
print("1. The problematic documents have been identified");
print("2. Consider deleting these documents if they cannot be fixed:");
problematicIds.forEach(function (id) {
  print("   db['generative-usages'].deleteOne({_id: ObjectId('" + id + "')})");
});
print(
  "3. Deploy the updated code with UTF-8 sanitization to prevent future issues"
);
print("4. Monitor logs for any UTF-8 related errors");
