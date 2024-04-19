import axios from "axios";

const api = axios.create({
  baseURL: "http://localhost:3000/graphql", // Replace with your API endpoint
  timeout: 10000, // Set timeout (optional)
  headers: {
    "Content-Type": "application/json",
    // Add any other default headers here
  },
});

export default api;
