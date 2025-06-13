#!/usr/bin/env node
// Example Node.js client for Generative API Router

const axios = require("axios");

const API_BASE = "http://localhost:8082";

async function chatCompletion(message) {
  try {
    const response = await axios.post(`${API_BASE}/v1/chat/completions`, {
      model: "any-model",
      messages: [{ role: "user", content: message }],
    });
    return response.data;
  } catch (error) {
    console.error("Error:", error.response?.data || error.message);
    return null;
  }
}

async function processFile(fileUrl, question, headers = null) {
  try {
    const content = [
      {
        type: "text",
        text: question
      },
      {
        type: "file_url",
        file_url: {
          url: fileUrl
        }
      }
    ];

    // Add custom headers if provided
    if (headers) {
      content[1].file_url.headers = headers;
    }

    const response = await axios.post(`${API_BASE}/v1/chat/completions`, {
      model: "document-analyzer",
      messages: [{ role: "user", content: content }],
    });
    return response.data;
  } catch (error) {
    console.error("Error:", error.response?.data || error.message);
    return null;
  }
}

async function processMultipleFiles(fileUrls, question) {
  try {
    const content = [{ type: "text", text: question }];
    
    // Add each file to the content
    fileUrls.forEach(url => {
      content.push({
        type: "file_url",
        file_url: { url: url }
      });
    });

    const response = await axios.post(`${API_BASE}/v1/chat/completions`, {
      model: "multi-file-analyzer",
      messages: [{ role: "user", content: content }],
    });
    return response.data;
  } catch (error) {
    console.error("Error:", error.response?.data || error.message);
    return null;
  }
}

// Example usage
(async () => {
  console.log("=== Basic Chat Example ===");
  const result = await chatCompletion("Hello, how are you?");
  if (result) console.log(JSON.stringify(result, null, 2));

  console.log("\n=== File Processing Example ===");
  // Example with Apple's research paper
  const fileResult = await processFile(
    "https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf",
    "Please provide a brief summary of this research paper."
  );
  if (fileResult) console.log(JSON.stringify(fileResult, null, 2));

  console.log("\n=== Multiple Files Example ===");
  // Example with multiple files
  const multiResult = await processMultipleFiles([
    "https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf",
    "https://example.com/another-document.pdf"  // Would need a real URL
  ], "Compare these two documents.");
  if (multiResult) console.log(JSON.stringify(multiResult, null, 2));
})();
