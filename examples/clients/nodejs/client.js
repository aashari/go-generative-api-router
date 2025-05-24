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
  }
}

// Example usage
(async () => {
  const result = await chatCompletion("Hello, how are you?");
  console.log(JSON.stringify(result, null, 2));
})();
